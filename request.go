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
	baseURL *url.URL

	PreviousURL *url.URL
	URL         *url.URL

	// Ctx is a context between a Request and a Response
	Ctx *Context
	// Depth is the number of the parents of the request
	Depth int

	abort bool
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

func (r *Request) String() string {
	urlJumping := r.URL.String()
	if r.PreviousURL != nil {
		urlJumping = r.PreviousURL.String() + " => " + urlJumping
	}

	return fmt.Sprintf("(%d) %s",
		r.Depth,
		urlJumping,
	)
}
