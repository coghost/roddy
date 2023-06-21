package roddy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"roddy/storage"

	"github.com/coghost/xbot"
	"github.com/coghost/xutil"
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

	c.wg = &sync.WaitGroup{}

	c.maxDepth = 0
	c.maxRequests = 0

	c.store = &storage.InMemoryStorage{}
	c.store.Init()

	c.highlightCount = 2
	c.highlightStyle = `box-shadow: 0 0 10px rgba(255,125,0,1), 0 0 20px 5px rgba(255,175,0,0.8), 0 0 30px 15px rgba(255,225,0,0.5);`

	c.baseDir = "/tmp/roddy"
	c.cookieDir = c.baseDir + "/cookies"
	c.cacheDir = c.baseDir + "/cache"

	c.lock = &sync.RWMutex{}
	c.ctx = context.Background()
}

// Wait returns when the collector jobs are finished
func (c *Collector) Wait() {
	c.wg.Wait()
}

// SetStorage overrides the default in-memory storage.
// Storage stores scraping related data like cookies and visited urls
func (c *Collector) SetStorage(s storage.Storage) error {
	if err := s.Init(); err != nil {
		return err
	}

	c.store = s

	return nil
}

// QuitOnTimeout blocks collector from close browser with 3(by default) seconds
//   - if async mode, you should call this directly
//   - if not async mode, this is enabled if c.quitInSeconds is not zero.
//
// about args
//   - when < 0, hang up until enter pressed
//   - when > 0, hang up in seconds
func (c *Collector) QuitOnTimeout(args ...int) {
	n := xutil.FirstOrDefaultArgs(3, args...)
	if n == 0 {
		return
	}

	if n < 0 {
		xutil.Pause()
	}

	Spin(n)

	c.Bot.Close()
}

func (c *Collector) Visit(URL string) error {
	return c.scrape(URL, 1, nil)
}

func (c *Collector) scrape(u string, depth int, ctx *Context) error {
	c.createBot()

	parsedURL, err := ParseUrl(u)
	if err != nil {
		return err
	}

	if err := c.requestCheck(parsedURL, depth); err != nil {
		err = c.handleIgnoredErrors(err)
		return err
	}

	c.wg.Add(1)

	if c.async {
		return c.asyncFetch(parsedURL, depth, ctx)
	}

	return c.fetch(parsedURL, depth, ctx)
}

func (c *Collector) asyncFetch(parsedURL *url.URL, depth int, ctx *Context) error {
	errChan := make(chan error, 1)

	go func() {
		err := c.fetch(parsedURL, depth, ctx)
		err = c.handleIgnoredErrors(err)

		if err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (c *Collector) fetch(URL *url.URL, depth int, ctx *Context) error {
	defer c.wg.Done()
	defer c.randomSleep()

	page := c.createPage()
	if c.async {
		defer c.pagePool.Put(page)
	}

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
		page:      page,
	}

	c.handleOnRequest(request)

	if request.abort {
		return nil
	}

	log.Debug().Str("request", request.String()).Msg("visiting")

	if e := page.Timeout(xbot.MediumToSec * time.Second).Navigate(URL.String()); e != nil {
		return e
	}

	err := page.Timeout(xbot.MediumToSec * time.Second).WaitLoad()
	if err != nil {
		c.handleOnError(nil, err, request, ctx)
		return err
	}

	response := &Response{
		Page:    page,
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

func (c *Collector) handleIgnoredErrors(err error) error {
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
				c.MustGoBack(resp.Page)
				continue
			}

			if i >= len(elems) {
				// elems may changed while we re-get all elements.
				continue
			}

			e := NewHTMLElement(resp, elems[i], cb.Selector, cbIndex)

			parent := fmt.Sprintf("%s: I-%d/%d", request.IDString(), i, count)

			txt := e.Target()
			if len(txt) > 32 {
				txt = xutil.TruncateString(e.Target(), 32) + "..."
			}

			target := fmt.Sprintf("#%d | %s[%d] | %s", request.Depth+1, cb.Selector, i, txt)

			msg := "spawn"
			if c.prevRequest != nil && c.prevRequest.ID > request.ID {
				msg = "recall"
			}

			// each time when switch from one request to another, we log and highlight it.
			if c.prevRequest == nil || c.prevRequest.ID != request.ID {
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
	err = c.handleIgnoredErrors(err)

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

func (c *Collector) UnmarshalRequest(r []byte) (*Request, error) {
	req := &serializableRequest{}

	err := json.Unmarshal(r, req)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	ctx := NewContext()
	for k, v := range req.Ctx {
		ctx.Put(k, v)
	}

	return &Request{
		URL:       u,
		Depth:     req.Depth,
		Ctx:       ctx,
		ID:        atomic.AddUint32(&c.requestCount, 1),
		collector: c,
	}, nil
}

func (c *Collector) randomSleep() {
	rd := time.Duration(0)
	if c.randomDelay != 0 {
		rd = time.Duration(rand.Int63n(int64(c.randomDelay)))
	}

	time.Sleep(c.delay + rd)
}
