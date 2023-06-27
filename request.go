package roddy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/coghost/xbot"
	"github.com/go-rod/rod"
	whatwgUrl "github.com/nlnwa/whatwg-url/url"
)

// RequestCallback is a type alias for OnRequest callback functions
type RequestCallback func(*Request)

type Request struct {
	// ID is the Unique identifier of the request
	ID uint32

	URL *url.URL

	// Ctx is a context between a Request and a Response
	Ctx *Context
	// Depth is the number of the parents of the request
	Depth int

	abort bool

	baseURL   *url.URL
	collector *Collector
	bot       *xbot.Bot
	page      *rod.Page
}

type serializableRequest struct {
	ID    uint32
	URL   string
	Depth int
	Ctx   map[string]interface{}
}

var urlParser = whatwgUrl.NewParser(whatwgUrl.WithPercentEncodeSinglePercentSign())

func (r *Request) New(URL string) (*Request, error) {
	u2, err := ParseUrl(URL)
	if err != nil {
		return nil, err
	}

	return &Request{
		URL:       u2,
		Ctx:       r.Ctx,
		ID:        atomic.AddUint32(&r.collector.requestCount, 1),
		collector: r.collector,
	}, nil
}

func (r *Request) AbsoluteURL(u string) string {
	if strings.HasPrefix(u, "#") {
		return ""
	}

	var base *url.URL
	if r.baseURL != nil {
		base = r.baseURL
	} else {
		base = r.URL
	}

	absURL, err := urlParser.ParseRef(base.String(), u)
	if err != nil {
		return ""
	}

	return absURL.Href(false)
}

func (r *Request) IDString() string {
	pg := r.bot.UniqueID
	if r.page != nil {
		pg = fmt.Sprintf("<%s:%s>", pg, r.page.String()[6:14])
	}

	return fmt.Sprintf("C-%d#%d(%s).R-%d", r.collector.ID, r.Depth, pg, r.ID)
}

func (r *Request) String() string {
	return fmt.Sprintf("%s | %s", r.IDString(), r.URL.String())
}

func (r *Request) Visit(URL string) error {
	return r.collector.scrape(r.AbsoluteURL(URL), r.Depth+1, r.Ctx)
}

func (r *Request) Do() error {
	return r.collector.scrape(r.URL.String(), r.Depth, r.Ctx)
}

// Marshal serializes the Request
func (r *Request) Marshal() ([]byte, error) {
	ctx := make(map[string]interface{})

	if r.Ctx != nil {
		r.Ctx.ForEach(func(k string, v interface{}) interface{} {
			ctx[k] = v
			return nil
		})
	}

	req := &serializableRequest{
		URL:   r.URL.String(),
		Depth: r.Depth,
		Ctx:   ctx,
		ID:    r.ID,
	}

	return json.Marshal(req)
}

func ParseUrl(URL string) (*url.URL, error) {
	u, err := urlParser.Parse(URL)
	if err != nil {
		return nil, err
	}

	u2, err := url.Parse(u.Href(false))
	if err != nil {
		return nil, err
	}

	return u2, err
}
