package roddy

import (
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// HTMLCallback is a type alias for OnHTML callback functions
type HTMLCallback func(e *HTMLElement)

type htmlCallbackContainer struct {
	Selector string
	Function HTMLCallback
}

type HTMLElement struct {
	Selector string

	Name string
	Text string

	attributes []html.Attribute

	DOM *goquery.Selection

	Request  *Request
	Response *Response

	Index int
}

func NewHTMLElement(resp *Response, s *goquery.Selection, n *html.Node, index int) *HTMLElement {
	return &HTMLElement{
		Name:     n.Data,
		Text:     goquery.NewDocumentFromNode(n).Text(),
		DOM:      s,
		Request:  resp.Request,
		Response: resp,
		Index:    index,

		attributes: n.Attr,
	}
}

func (e *HTMLElement) Attr(k string) string {
	for _, a := range e.attributes {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}
