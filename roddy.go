package roddy

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"net/url"
	"regexp"
	"sync"
	"time"

	"roddy/storage"

	"github.com/coghost/xbot"
	"github.com/coghost/xlog"
	"github.com/coghost/xpretty"
	"github.com/remeh/sizedwaitgroup"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	_cap = 4
)

func (c *Collector) Init() {
	// TODO: remove this
	xlog.InitLog(xlog.WithLevel(zerolog.InfoLevel), xlog.WithNoColor(false))
	xpretty.Initialize(xpretty.WithNoColor(false))

	c.userAgent = xbot.UA
	c.headless = false

	c.store = &storage.InMemoryStorage{}
	c.store.Init()

	c.wg = sizedwaitgroup.New(4)
	c.lock = &sync.RWMutex{}
	c.ctx = context.Background()
}

func (c *Collector) InitDefaultBot() {
	if c.pauseBeforeQuit {
		c.headless = false
	}

	bof := []xbot.BotOptFunc{
		xbot.BotSpawn(false),
		xbot.BotScreen(-2560),
		xbot.BotHeadless(c.headless),
		xbot.BotUserAgent(c.userAgent),
	}

	c.Bot = xbot.NewBot(bof...)
}

func (c *Collector) Visit(URL string) error {
	defer func() {
		if c.pauseBeforeQuit {
			fmt.Println("sleep an hour, press Ctrl+C to quit.")
			time.Sleep(time.Hour)
		}
	}()

	if c.Bot.Brw == nil {
		log.Debug().Msg("no bot existed, create bot resources...")
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
		return err
	}

	return c.fetch(parsedURL, depth, ctx)
}

func (c *Collector) fetch(URL *url.URL, depth int, ctx *Context) error {
	if ctx == nil {
		ctx = NewContext()
	}

	request := &Request{
		URL:   URL,
		Ctx:   ctx,
		Depth: depth,

		PreviousURL: c.previousURL,
	}

	c.previousURL = URL

	c.handleOnRequest(request)

	if request.abort {
		return nil
	}

	log.Debug().Str("url", URL.String()).Msg("fetching")

	c.Bot.GetPage(URL.String())

	response := &Response{
		Page:    c.Bot.Pg,
		Request: request,
		Ctx:     ctx,
	}

	c.handleOnResponse(response)

	err := c.handleOnHTML(response)
	if err != nil {
		c.handleOnError(response, err, request, ctx)
	}

	return err
}

func (c *Collector) requestCheck(parsedURL *url.URL, depth int) error {
	if c.maxDepth > 0 && c.maxDepth < depth {
		return ErrMaxDepth
	}

	if c.maxRequests > 0 && c.maxRequests <= c.requestCount {
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
OnError
OnResponse
- OnHTML
OnScraped
**/

// OnRequest
func (c *Collector) OnRequest(f RequestCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.requestCallbacks == nil {
		c.requestCallbacks = make([]RequestCallback, 0, _cap)
	}

	c.requestCallbacks = append(c.requestCallbacks, f)
}

func (c *Collector) OnHTML(selector string, f HTMLCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.htmlCallbacks == nil {
		c.htmlCallbacks = make([]*htmlCallbackContainer, 0, _cap)
	}

	c.htmlCallbacks = append(c.htmlCallbacks, &htmlCallbackContainer{
		Selector: selector,
		Function: f,
	})
}

// OnResponse handle on response.
func (c *Collector) OnResponse(f ResponseCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.responseCallbacks == nil {
		c.responseCallbacks = make([]ResponseCallback, 0, _cap)
	}

	c.responseCallbacks = append(c.responseCallbacks, f)
}

// OnError registers a function. Function will be executed if an error
// occurs during the HTTP request.
func (c *Collector) OnError(f ErrorCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.errorCallbacks == nil {
		c.errorCallbacks = make([]ErrorCallback, 0, _cap)
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

func (c *Collector) handleOnHTML(resp *Response) error {
	if len(c.htmlCallbacks) == 0 {
		return nil
	}

	for cbIndex, cb := range c.htmlCallbacks {
		elems := resp.Page.MustElements(cb.Selector)
		for _, elem := range elems {
			e := NewHTMLElement(resp, elem, cb.Selector, cbIndex)
			cb.Function(e)
		}
	}

	return nil
}

func (c *Collector) handleOnError(response *Response, err error, request *Request, ctx *Context) error {
	if err == nil {
		return nil
	}

	if response == nil {
		response = &Response{
			Request: request,
			Ctx:     ctx,
		}
	}

	for _, f := range c.errorCallbacks {
		f(response, err)
	}

	return nil
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
