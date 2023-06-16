package roddy

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/url"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"roddy/storage"

	"github.com/coghost/xbot"
	"github.com/coghost/xpretty"
	"github.com/coghost/xutil"
	"github.com/gookit/goutil/arrutil"
	"github.com/rs/zerolog/log"
)

const (
	_capacity = 4

	_waitGroupSize = 4
)

var collectorCounter uint32

func (c *Collector) Init() {
	c.ID = atomic.AddUint32(&collectorCounter, 1)
	c.userAgent = xbot.UA
	c.headless = false

	c.maxDepth = 0
	c.maxRequests = 0

	c.store = &storage.InMemoryStorage{}
	c.store.Init()

	c.highlightCount = 2
	c.highlightStyle = `box-shadow: 0 0 10px rgba(255,125,0,1), 0 0 20px 5px rgba(255,175,0,0.8), 0 0 30px 15px rgba(255,225,0,0.5);`

	c.lock = &sync.RWMutex{}
	c.ctx = context.Background()
}

func (c *Collector) InitDefaultBot() {
	proxy := ""

	if len(c.proxies) != 0 {
		proxy = arrutil.RandomOne(c.proxies)
	}

	bof := []xbot.BotOptFunc{
		xbot.BotSpawn(false),
		xbot.BotScreen(0),
		xbot.BotHeadless(c.headless),
		xbot.BotUserAgent(c.userAgent),
		xbot.BotProxyServer(proxy),
	}

	c.Bot = xbot.NewBot(bof...)
}

// HangUp will sleep an hour before quit, so we can check out what happends
func (c *Collector) HangUp() {
	xutil.Pause("")
}

func (c *Collector) MustGoBack() {
	if c.maxDepth == 0 {
		return
	}

	err := c.Bot.Pg.NavigateBack()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot go back")
	}

	c.Bot.Pg.MustWaitLoad()
}

// HangUpInSeconds hangs up browser within 3(by default) seconds
func (c *Collector) HangUpInSeconds(args ...int) {
	n := xutil.FirstOrDefaultArgs(3, args...)
	xpretty.YellowPrintf("quit in %d seconds ...\n", n)

	time.Sleep(time.Second * time.Duration(n))
}

// HangUpHourly sleeps an hour, used when running test
func (c *Collector) HangUpHourly() {
	xpretty.YellowPrintf("quit in one hour ...\n")
	time.Sleep(time.Hour)
}

func (c *Collector) Visit(URL string) error {
	if c.Bot.Brw == nil {
		log.Trace().Msg("no bot found, create bot")
		xbot.Spawn(c.Bot)
	}

	return c.scrape(URL, 1, nil)
}

func (c *Collector) scrape(u string, depth int, ctx *Context) error {
	parsedURL, err := str2URL(u)
	if err != nil {
		return err
	}

	if err := c.requestCheck(parsedURL, depth); err != nil {
		err = c.checkIgnoredError(err)
		return err
	}

	return c.fetch(parsedURL, depth, ctx)
}

func (c *Collector) checkIgnoredError(err error) error {
	for _, ie := range c.ignoredErrors {
		if err == ie || errors.Unwrap(err) == ie {
			return nil
		}
	}

	var ave *AlreadyVisitedError
	if c.ignoreVistedError && errors.As(err, &ave) {
		return nil
	}

	return err
}

func (c *Collector) fetch(URL *url.URL, depth int, ctx *Context) error {
	if ctx == nil {
		ctx = NewContext()
	}

	rid := atomic.AddUint32(&c.requestCount, 1)
	request := &Request{
		ID:    rid,
		URL:   URL,
		Ctx:   ctx,
		Depth: depth,

		collector: c,
	}

	c.previousURL = URL

	c.handleOnRequest(request)

	if request.abort {
		return nil
	}

	log.Debug().Str("request", request.String()).Msg("visiting")

	err := c.Bot.GetPageE(URL.String())
	if err != nil {
		c.handleOnError(nil, err, request, ctx)
		return err
	}

	response := &Response{
		Page:    c.Bot.Pg,
		Request: request,
		Ctx:     ctx,
	}

	c.responseCount++
	c.handleOnResponse(response)

	err = c.handleOnSerp(response)
	if err != nil {
		err = c.handleOnError(response, err, request, ctx)
		return err
	}

	err = c.handleOnHTML(response)
	if err != nil {
		err = c.handleOnError(response, err, request, ctx)
		return err
	}

	c.handleOnScraped(response)

	return err
}

func (c *Collector) requestCheck(parsedURL *url.URL, depth int) error {
	if c.maxDepth > 0 && c.maxDepth < depth {
		return ErrMaxDepth
	}

	if c.maxRequests > 0 && c.requestCount >= c.maxRequests {
		return ErrMaxRequests
	}

	if err := c.checkFilters(parsedURL, parsedURL.Hostname()); err != nil {
		return err
	}

	if err := c.checkVistedStatus(parsedURL); err != nil {
		return err
	}

	return nil
}

func (c *Collector) checkFilters(parsedURL *url.URL, domain string) error {
	u := parsedURL.String()

	if len(c.disallowedURLFilters) > 0 {
		if isMatchingFilter(c.disallowedURLFilters, []byte(u)) {
			return ErrForbiddenURL
		}
	}

	if len(c.urlFilters) > 0 {
		if !isMatchingFilter(c.urlFilters, []byte(u)) {
			return ErrNoURLFiltersMatch
		}
	}

	if !c.isDomainAllowed(domain) {
		return ErrForbiddenDomain
	}

	return nil
}

func (c *Collector) checkVistedStatus(parsedURL *url.URL) error {
	if c.allowURLRevisit {
		return nil
	}

	u := parsedURL.String()
	uHash := requestHash(u, nil)

	visited, err := c.store.IsVisited(uHash)
	if err != nil {
		return err
	}

	if visited {
		return &AlreadyVisitedError{parsedURL}
	}

	return c.store.Visited(uHash)
}

func (c *Collector) isDomainAllowed(domain string) bool {
	for _, d2 := range c.disallowedDomains {
		if d2 == domain {
			return false
		}
	}

	if c.allowedDomains == nil || len(c.allowedDomains) == 0 {
		return true
	}

	for _, d2 := range c.allowedDomains {
		if d2 == domain {
			return true
		}
	}

	return false
}

// func (c *Collector) VisitByClick(selector string) error {
// }

/** Callbacks **/

/**
- OnRequest
- OnResponse
- OnSerp
- OnHTML
- OnScraped
- OnError
**/

// OnRequest
func (c *Collector) OnRequest(f RequestCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.requestCallbacks == nil {
		c.requestCallbacks = make([]RequestCallback, 0, _capacity)
	}

	c.requestCallbacks = append(c.requestCallbacks, f)
}

func (c *Collector) OnSerp(selector string, f SerpCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.serpCallbacks == nil {
		c.serpCallbacks = make([]*serpCallbackContainer, 0, _capacity)
	}

	c.serpCallbacks = append(c.serpCallbacks, &serpCallbackContainer{
		Selector: selector,
		Function: f,
	})
}

func (c *Collector) OnHTML(selector string, f HTMLCallback, opts ...OnHTMLOptionFunc) {
	c.lock.Lock()
	defer c.lock.Unlock()

	opt := OnHTMLOptions{deferFunc: func() {}}
	bindOnHTMLOptions(&opt, opts...)

	if c.htmlCallbacks == nil {
		c.htmlCallbacks = make([]*htmlCallbackContainer, 0, _capacity)
	}

	c.htmlCallbacks = append(c.htmlCallbacks, &htmlCallbackContainer{
		Selector:  selector,
		Function:  f,
		DeferFunc: opt.deferFunc,
	})
}

// OnResponse handle on response.
func (c *Collector) OnResponse(f ResponseCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.responseCallbacks == nil {
		c.responseCallbacks = make([]ResponseCallback, 0, _capacity)
	}

	c.responseCallbacks = append(c.responseCallbacks, f)
}

// OnError registers a function. Function will be executed if an error
// occurs during the HTTP request.
func (c *Collector) OnError(f ErrorCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.errorCallbacks == nil {
		c.errorCallbacks = make([]ErrorCallback, 0, _capacity)
	}

	c.errorCallbacks = append(c.errorCallbacks, f)
}

// OnHTMLDetach deregister a function. Function will not be execute after detached
func (c *Collector) OnHTMLDetach(goquerySelector string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	deleteIdx := -1

	for i, cc := range c.htmlCallbacks {
		if cc.Selector == goquerySelector {
			deleteIdx = i
			break
		}
	}

	if deleteIdx != -1 {
		c.htmlCallbacks = append(c.htmlCallbacks[:deleteIdx], c.htmlCallbacks[deleteIdx+1:]...)
	}
}

// OnScraped registers a function. Function will be executed after
// OnHTML, as a final part of the scraping.
func (c *Collector) OnScraped(f ScrapedCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.scrapedCallbacks == nil {
		c.scrapedCallbacks = make([]ScrapedCallback, 0, 4)
	}

	c.scrapedCallbacks = append(c.scrapedCallbacks, f)
}

func (c *Collector) handleOnRequest(r *Request) {
	for _, f := range c.requestCallbacks {
		f(r)
	}
}

func (c *Collector) handleOnResponse(r *Response) {
	for _, f := range c.responseCallbacks {
		f(r)
	}
}

func (c *Collector) handleOnSerp(resp *Response) error {
	if len(c.serpCallbacks) == 0 {
		return nil
	}

	for cbIndex, cb := range c.serpCallbacks {
		pg := resp.Page

		elem, err := pg.Element(cb.Selector)
		if err != nil {
			continue
		}

		c.Bot.BindRoot(elem)
		defer c.Bot.ResetRoot()

		e := NewSerpElement(resp, elem, cb.Selector, cbIndex)
		cb.Function(e)
	}

	return nil
}

func (c *Collector) handleOnHTML(resp *Response) error {
	if len(c.htmlCallbacks) == 0 {
		return nil
	}

	finalDepth := resp.Request.Depth >= c.maxDepth
	request := resp.Request

	for cbIndex, cb := range c.htmlCallbacks {
		// after current page's elements are handled, go back
		if cb.DeferFunc != nil {
			defer cb.DeferFunc()
		}

		pg := resp.Page
		// count := len(pg.MustElements(cb.Selector))

		elems, err := pg.Elements(cb.Selector)
		if err != nil {
			return err
		}

		count := len(elems)
		if count == 0 {
			return fmt.Errorf("%s: %w", request.String(), ErrNoElemFound)
		}

		// this will skip page with depth of maxDepth
		if c.skipOnHTMLOfMaxDepth && finalDepth {
			msg := fmt.Sprintf("[CID-%d]: %s", c.ID, request.String())
			log.Debug().Msgf("max depth reached, skip: %s", msg)

			continue
		}

		for i := 0; i < count; i++ {
			// WARN: elems are not accessable after page is changed, we have to re-get all elements, then get correct elem by index.
			elems, err := pg.Elements(cb.Selector)
			if err != nil || len(elems) == 0 {
				c.MustGoBack()
				continue
			}

			e := NewHTMLElement(resp, elems[i], cb.Selector, cbIndex)

			parent := fmt.Sprintf("%s: I-%d/%d", request.IDString(), i, count)
			target := fmt.Sprintf("#%d | %s[%d] | %s", request.Depth+1, cb.Selector, i, e.Target())

			msg := "spawn"
			if c.prevRequest != nil && c.prevRequest.ID > request.ID {
				msg = "recall"
			}

			// each time when switch from one request to another, we log and highlight it.
			if c.prevRequest == nil || c.prevRequest.ID != request.ID {
				log.Debug().Str("with", target).Str("from", parent).Msg(msg)
				e.Focus(c.highlightCount, c.highlightStyle)
				c.prevRequest = request
			}

			log.Trace().Str("with", target).Str("from", parent).Msg(msg)
			cb.Function(e)
		}
	}

	return nil
}

func (c *Collector) handleOnError(response *Response, err error, request *Request, ctx *Context) error {
	err = c.checkIgnoredError(err)

	if err == nil {
		return nil
	}

	if response == nil {
		response = &Response{
			Request: request,
			Ctx:     ctx,
		}
	}

	if response.Request == nil {
		response.Request = request
	}

	if response.Ctx == nil {
		response.Ctx = request.Ctx
	}

	for _, f := range c.errorCallbacks {
		f(response, err)
	}

	return err
}

func (c *Collector) handleOnScraped(r *Response) {
	for _, f := range c.scrapedCallbacks {
		f(r)
	}
}

func isMatchingFilter(fs []*regexp.Regexp, d []byte) bool {
	for _, r := range fs {
		if r.Match(d) {
			return true
		}
	}

	return false
}

func str2URL(u string) (*url.URL, error) {
	parsedWhatwgURL, err := urlParser.Parse(u)
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(parsedWhatwgURL.Href(false))
	if err != nil {
		return nil, err
	}

	return parsedURL, nil
}

func normalizeURL(u string) string {
	parsed, err := urlParser.Parse(u)
	if err != nil {
		return u
	}

	return parsed.String()
}

func requestHash(url string, body io.Reader) uint64 {
	h := fnv.New64a()
	// reparse the url to fix ambiguities such as
	// "http://example.com" vs "http://example.com/"
	io.WriteString(h, normalizeURL(url))

	if body != nil {
		io.Copy(h, body)
	}

	return h.Sum64()
}
