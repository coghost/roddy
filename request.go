package roddy

import (
	"fmt"
	"net/url"
	"strings"

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
}

var urlParser = whatwgUrl.NewParser(whatwgUrl.WithPercentEncodeSinglePercentSign())

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
	return fmt.Sprintf("C-%d#%d.R-%d", r.collector.ID, r.Depth, r.ID)
}

func (r *Request) String() string {
	return fmt.Sprintf("%s | %s", r.IDString(), r.URL.String())
}

func (r *Request) Visit(URL string) error {
	return r.collector.scrape(r.AbsoluteURL(URL), r.Depth+1, r.Ctx)
}
