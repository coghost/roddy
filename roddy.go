package roddy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"roddy/storage"

	"github.com/PuerkitoBio/goquery"
	"github.com/coghost/xbot"
	"github.com/coghost/xutil"
	"github.com/go-rod/rod"
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

	c.baseDir = "/tmp/roddy"
	c.cookieDir = c.baseDir + "/cookies"
	c.cacheDir = c.baseDir + "/cache"

	c.wg = &sync.WaitGroup{}
	c.lock = &sync.RWMutex{}
	c.ctx = context.Background()
}

func (c *Collector) registerCtrlC() {
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-ch
		os.Exit(1)
	}()
}

// String is the text representation of the collector.
// It contains useful debug information about the collector's internals
func (c *Collector) String() string {
	return fmt.Sprintf(
		"Requests made: %d (%d responses) | Callbacks: OnRequest: %d, OnHTML: %d, OnResponse: %d, OnError: %d",
		atomic.LoadUint32(&c.requestCount),
		atomic.LoadUint32(&c.responseCount),
		len(c.requestCallbacks),
		len(c.dataCallbacks),
		len(c.responseCallbacks),
		len(c.errorCallbacks),
	)
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

	// c.Bot.Close()
	SleepWithSpin(n)
}

func (c *Collector) Visit(URL string) error {
	return c.scrape(URL, 1, nil)
}

func (c *Collector) getParsedURL(u string, depth int) (*url.URL, error) {
	if u == BlankPagePlaceholder {
		return nil, nil
	}

	parsedURL, err := ParseUrl(u)
	if err != nil {
		return nil, err
	}

	if err := c.requestCheck(parsedURL, depth); err != nil {
		err = c.handleIgnoredErrors(err)
		return nil, err
	}

	return parsedURL, nil
}

func (c *Collector) scrape(u string, depth int, ctx *Context) error {
	parsedURL, err := c.getParsedURL(u, depth)
	if err != nil {
		return err
	}

	if c.async {
		c.wg.Add(1)
		return c.asyncFetch(parsedURL, depth, ctx)
	}

	return c.fetch(parsedURL, depth, ctx)
}

func (c *Collector) asyncFetch(parsedURL *url.URL, depth int, ctx *Context) error {
	errChan := make(chan error, 1)

	go func() {
		defer c.wg.Done()

		c.waitChan <- true
		defer func(c *Collector) {
			c.randomSleep()
			<-c.waitChan
		}(c)

		err := c.fetch(parsedURL, depth, ctx)
		err = c.handleIgnoredErrors(err)

		if err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		log.Error().Err(err).Msg("got err")
		return err
	default:
		return nil
	}
}

func (c *Collector) fetch(URL *url.URL, depth int, ctx *Context) error {
	bot, page := c.createPage()

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
		bot:       bot,
		page:      page,
	}

	c.handleOnRequest(request)

	if request.abort {
		return nil
	}

	response, err := c.MustGet(request, page, URL, depth)
	if err != nil {
		return c.handleOnError(nil, err, request, ctx)
	}

	atomic.AddUint32(&c.responseCount, 1)

	response.Ctx = ctx

	c.handleOnResponse(response)

	err = c.handleOnHTML(response)
	if err != nil {
		return c.handleOnError(response, err, request, ctx)
	}

	err = c.handleOnData(response)
	if err != nil {
		return c.handleOnError(response, err, request, ctx)
	}

	if c.maxResponses > 0 && c.responseCount >= c.maxResponses {
		return ErrMaxResponses
	}

	err = c.handleMaxPageNum()
	if err != nil {
		return err
	}

	err = c.handleOnPaging(response)
	if err != nil {
		return c.handleOnError(response, err, request, ctx)
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
- OnHTML
- OnSerp
- OnScraped

- OnError
**/

// OnRequest
func (c *Collector) OnRequest(f RequestCallback) {
	c.lock.Lock()

	if c.requestCallbacks == nil {
		c.requestCallbacks = make([]RequestCallback, 0, _capacity)
	}

	c.requestCallbacks = append(c.requestCallbacks, f)
	c.lock.Unlock()
}

// OnResponse handle on response.
func (c *Collector) OnResponse(f ResponseCallback) {
	c.lock.Lock()

	if c.responseCallbacks == nil {
		c.responseCallbacks = make([]ResponseCallback, 0, _capacity)
	}

	c.responseCallbacks = append(c.responseCallbacks, f)

	c.lock.Unlock()
}

func (c *Collector) OnHTML(selector string, f HTMLCallback, opts ...CallbackOptionFunc) {
	c.lock.Lock()

	opt := CallbackOptions{deferFunc: func(p *rod.Page) {}}
	bindCallbackOptions(&opt, opts...)

	if c.htmlCallbacks == nil {
		c.htmlCallbacks = make([]*htmlCallbackContainer, 0, _capacity)
	}

	c.htmlCallbacks = append(c.htmlCallbacks, &htmlCallbackContainer{
		Selector:  selector,
		Function:  f,
		DeferFunc: opt.deferFunc,
	})

	c.lock.Unlock()
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

		log.Info().Str("selector", goquerySelector).Msg("detached handler on")
	}
}

func (c *Collector) OnData(selector string, f DataCallback) {
	c.lock.Lock()

	if c.dataCallbacks == nil {
		c.dataCallbacks = make([]*dataCallbackContainer, 0, _capacity)
	}

	c.dataCallbacks = append(c.dataCallbacks, &dataCallbackContainer{
		Selector: selector,
		Function: f,
	})
	c.lock.Unlock()
}

func (c *Collector) OnPaging(selector string, f HTMLCallback, opts ...CallbackOptionFunc) {
	c.lock.Lock()

	opt := CallbackOptions{deferFunc: func(p *rod.Page) {}}
	bindCallbackOptions(&opt, opts...)

	if c.pagingCallbacks == nil {
		c.pagingCallbacks = make([]*htmlCallbackContainer, 0, _capacity)
	}

	c.pagingCallbacks = append(c.pagingCallbacks, &htmlCallbackContainer{
		Selector:  selector,
		Function:  f,
		DeferFunc: opt.deferFunc,
	})

	c.lock.Unlock()
}

// OnError registers a function. Function will be executed if an error
// occurs during the HTTP request.
func (c *Collector) OnError(f ErrorCallback) {
	c.lock.Lock()

	if c.errorCallbacks == nil {
		c.errorCallbacks = make([]ErrorCallback, 0, _capacity)
	}

	c.errorCallbacks = append(c.errorCallbacks, f)

	c.lock.Unlock()
}

// OnScraped registers a function. Function will be executed after
// OnHTML, as a final part of the scraping.
func (c *Collector) OnScraped(f ScrapedCallback) {
	c.lock.Lock()

	if c.scrapedCallbacks == nil {
		c.scrapedCallbacks = make([]ScrapedCallback, 0, 4)
	}

	c.scrapedCallbacks = append(c.scrapedCallbacks, f)

	c.lock.Unlock()
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

func (c *Collector) handleOnData(resp *Response) error {
	if len(c.dataCallbacks) == 0 {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer([]byte(resp.Page.MustHTML())))
	if err != nil {
		return err
	}

	// try parse base from
	if href, found := doc.Find("base[href]").Attr("href"); found {
		u, err := urlParser.ParseRef(resp.Request.URL.String(), href)
		if err == nil {
			baseURL, err := url.Parse(u.Href(false))
			if err == nil {
				resp.Request.baseURL = baseURL
			}
		}
	}

	for _, cb := range c.dataCallbacks {
		cbIndex := 0

		doc.Find(cb.Selector).Each(func(_ int, s *goquery.Selection) {
			for _, n := range s.Nodes {
				e := NewHTMLElement(resp, s, n, cbIndex)
				cbIndex++
				cb.Function(e)
			}
		})
	}

	return nil
}

func (c *Collector) handleOnHTML(resp *Response) error {
	return c.handleOnSerp(resp, c.htmlCallbacks)
}

func (c *Collector) handleOnPaging(resp *Response) error {
	return c.handleOnSerp(resp, c.pagingCallbacks)
}

func (c *Collector) handleOnSerp(resp *Response, callbacks []*htmlCallbackContainer) error {
	if len(callbacks) == 0 {
		return nil
	}

	finalDepth := resp.Request.Depth >= c.maxDepth
	request := resp.Request

	for cbIndex, cb := range callbacks {
		// after current page's elements are handled, go back
		if cb.DeferFunc != nil {
			defer cb.DeferFunc(resp.Page)
		}

		bot := xbot.NewBotWithPage(resp.Page)

		elem := bot.GetElem(cb.Selector)
		if elem == nil {
			return fmt.Errorf("%s: %s: %w", request.String(), cb.Selector, ErrNoElemFound)
		}

		count := len(bot.GetElems(cb.Selector))

		// this will skip page with depth of maxDepth
		if c.skipOnHTMLOfMaxDepth && finalDepth {
			msg := fmt.Sprintf("[CID-%d]: %s", c.ID, request.String())
			log.Debug().Msgf("max depth reached, skip: %s", msg)

			continue
		}

		for i := 0; i < count; i++ {
			// WARN: elems are not accessable after page is changed, we have to re-get all elements, then get correct elem by index.
			elem := bot.GetElem(cb.Selector)
			if elem == nil {
				c.MustGoBack(bot.Pg)
				continue
			}

			elems := bot.GetElems(cb.Selector)

			if i >= len(elems) {
				// elems may changed while we re-get all elements.
				continue
			}

			e := NewSerpElement(resp, elems[i], cb.Selector, cbIndex)

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

			err := cb.Function(e)
			if err != nil {
				return err
			}
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
