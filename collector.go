package roddy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sync"

	"roddy/storage"

	"github.com/coghost/xbot"
)

type Collector struct {
	// Bot can be called out of collector
	Bot *xbot.Bot

	// userAgent is the User-Agent string used by Bot
	userAgent string
	// headless is whether using headless mode or not,
	// it will be force set to false, when pauseBeforeQuit is set to true.
	headless bool

	ctx context.Context

	// maxDepth limits the recursion depth of visited URLs.
	// Set it to 0 for infinite recursion (default).
	maxDepth int
	// maxRequests limit the number of requests done by the instance.
	// Set it to 0 for infinite requests (default).
	maxRequests uint32

	// allowedDomains is a domain whitelist.
	// Leave it blank to allow any domains to be visited
	allowedDomains []string
	// disallowedDomains is a domain blacklist.
	disallowedDomains []string
	// disallowedURLFilters is a list of regular expressions which restricts
	// visiting URLs. If any of the rules matches to a URL the
	// request will be stopped. disallowedURLFilters will
	// be evaluated before URLFilters
	// Leave it blank to allow any URLs to be visited
	disallowedURLFilters []*regexp.Regexp
	// Leave it blank to allow any URLs to be visited
	urlFilters []*regexp.Regexp

	// allowURLRevisit allows multiple downloads of the same URL
	allowURLRevisit bool

	// store is used to identify if URL is visited or not
	store storage.Storage
	// previousURL is the url before visiting current one
	previousURL *url.URL

	htmlCallbacks     []*htmlCallbackContainer
	requestCallbacks  []RequestCallback
	responseCallbacks []ResponseCallback
	errorCallbacks    []ErrorCallback

	ignoredErrors []error

	requestCount  uint32
	responseCount uint32

	lock *sync.RWMutex
}

// AlreadyVisitedError is the error type for already visited URLs.
//
// It's returned synchronously by Visit when the URL passed to Visit
// is already visited.
//
// When already visited URL is encountered after following
// redirects, this error appears in OnError callback, and if Async
// mode is not enabled, is also returned by Visit.
type AlreadyVisitedError struct {
	// Destination is the URL that was attempted to be visited.
	// It might not match the URL passed to Visit if redirect
	// was followed.
	Destination *url.URL
}

// Error implements error interface.
func (e *AlreadyVisitedError) Error() string {
	return fmt.Sprintf("%q already visited", e.Destination)
}

var (
	// ErrForbiddenDomain is the error thrown if visiting
	// a domain which is not allowed in AllowedDomains
	ErrForbiddenDomain = errors.New("Forbidden domain")
	// ErrMaxDepth is the error type for exceeding max depth
	ErrMaxDepth = errors.New("Max depth limit reached")
	// ErrForbiddenURL is the error thrown if visiting
	// a URL which is not allowed by URLFilters
	ErrForbiddenURL = errors.New("ForbiddenURL")

	// ErrNoURLFiltersMatch is the error thrown if visiting
	// a URL which is not allowed by URLFilters
	ErrNoURLFiltersMatch = errors.New("No URLFilters match")
	// ErrMaxRequests is the error returned when exceeding max requests
	ErrMaxRequests = errors.New("Max Requests limit reached")
)

func NewCollector(options ...CollectorOption) *Collector {
	c := &Collector{}
	// default settings
	c.Init()
	// bind options from args in
	bindOptions(c, options...)
	// finally setup bot
	c.InitDefaultBot()

	return c
}

type CollectorOption func(*Collector)

func bindOptions(c *Collector, options ...CollectorOption) {
	for _, f := range options {
		f(c)
	}
}

func UserAgent(ua string) CollectorOption {
	return func(c *Collector) {
		c.userAgent = ua
	}
}

func Headless(b bool) CollectorOption {
	return func(c *Collector) {
		c.headless = b
	}
}

func AllowURLRevisit(b bool) CollectorOption {
	return func(c *Collector) {
		c.allowURLRevisit = b
	}
}

// MaxDepth limits the recursion depth of visited URLs.
func MaxDepth(depth int) CollectorOption {
	return func(c *Collector) {
		c.maxDepth = depth
	}
}

// MaxRequests limit the number of requests done by the instance.
// Set it to 0 for infinite requests (default).
func MaxRequests(max uint32) CollectorOption {
	return func(c *Collector) {
		c.maxRequests = max
	}
}

// AllowedDomains sets the domain whitelist used by the Collector.
func AllowedDomains(domains ...string) CollectorOption {
	return func(c *Collector) {
		c.allowedDomains = domains
	}
}

// AllowedDomains sets the domain whitelist used by the Collector.
func DisallowedDomains(domains ...string) CollectorOption {
	return func(c *Collector) {
		c.disallowedDomains = domains
	}
}

// DisallowedURLFilters sets the list of regular expressions which restricts
// visiting URLs. If any of the rules matches to a URL the request will be stopped.
func DisallowedURLFilters(filters ...*regexp.Regexp) CollectorOption {
	return func(c *Collector) {
		c.disallowedURLFilters = filters
	}
}

// URLFilters sets the list of regular expressions which restricts
// visiting URLs. If any of the rules matches to a URL the request won't be stopped.
func URLFilters(filters ...*regexp.Regexp) CollectorOption {
	return func(c *Collector) {
		c.urlFilters = filters
	}
}

func IgnoredErrors(errs ...error) CollectorOption {
	return func(c *Collector) {
		c.ignoredErrors = errs
	}
}
