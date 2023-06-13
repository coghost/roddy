package roddy

import (
	"bytes"
	"context"
	"hash/fnv"
	"io"
	"net/url"
	"regexp"
	"sync"
	"time"

	"roddy/storage"

	"github.com/PuerkitoBio/goquery"
	"github.com/coghost/xbot"
	"github.com/coghost/xlog"
	"github.com/coghost/xpretty"
	"github.com/coghost/xutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	_capacity = 4

	_waitGroupSize = 4
)

func (c *Collector) Init() {
	c.initDefaultPrettyMode()
	c.userAgent = xbot.UA
	c.headless = false

	c.maxDepth = 0
	c.maxRequests = 0

	c.store = &storage.InMemoryStorage{}
	c.store.Init()

	c.lock = &sync.RWMutex{}
	c.ctx = context.Background()
}

func (c *Collector) InitDefaultBot() {
	bof := []xbot.BotOptFunc{
		xbot.BotSpawn(false),
		xbot.BotScreen(0),
		xbot.BotHeadless(c.headless),
		xbot.BotUserAgent(c.userAgent),
	}

	c.Bot = xbot.NewBot(bof...)
}

// HangUp will sleep an hour before quit, so we can check out what happends
func (c *Collector) HangUp() {
	xutil.Pause("")
}

func (c *Collector) GoBack() {
	c.Bot.Pg.MustNavigateBack()
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

func (c *Collector) initDefaultPrettyMode() {
	xlog.InitLog(xlog.WithLevel(zerolog.InfoLevel), xlog.WithNoColor(false), xlog.WithCaller(true))
	xpretty.Initialize(xpretty.WithNoColor(false))
}

func (c *Collector) Visit(URL string) error {
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
		collector:   c,
	}

	c.previousURL = URL

	c.handleOnRequest(request)

	if request.abort {
		return nil
	}

	log.Debug().Str("url", URL.String()).Msg("fetching")

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

	c.handleOnResponse(response)

	err = c.handleOnHTML(response)
	if err != nil {
		c.handleOnError(response, err, request, ctx)
	}

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
		c.requestCallbacks = make([]RequestCallback, 0, _capacity)
	}

	c.requestCallbacks = append(c.requestCallbacks, f)
}

func (c *Collector) OnHTML(selector string, f HTMLCallback) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.htmlCallbacks == nil {
		c.htmlCallbacks = make([]*htmlCallbackContainer, 0, _capacity)
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

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer([]byte(resp.Page.MustHTML())))
	if err != nil {
		return err
	}

	for _, cc := range c.htmlCallbacks {
		i := 0

		doc.Find(cc.Selector).Each(func(_ int, s *goquery.Selection) {
			for _, n := range s.Nodes {
				e := NewHTMLElement(resp, s, n, i)
				i++

				cc.Function(e)
			}
		})
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
