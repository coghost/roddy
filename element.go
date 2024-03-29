package roddy

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

type DataElement struct {
	TagName string
	Index   int

	// Request is the request object of the element's HTML document
	Request *Request
	// Response is the Response object of the element's HTML document
	Response *Response

	// DOM is the goquery parsed DOM object of the page. DOM is relative
	// to the current HTMLElement
	DOM  *goquery.Selection
	node *html.Node
}

func NewHTMLElement(resp *Response, s *goquery.Selection, n *html.Node, index int) *DataElement {
	return &DataElement{
		TagName: n.Data,
		Index:   index,

		Request:  resp.Request,
		Response: resp,

		DOM:  s,
		node: n,
	}
}

func (e *DataElement) Attr(k string) string {
	for _, a := range e.node.Attr {
		if a.Key == k {
			return a.Val
		}
	}

	return ""
}

func (e *DataElement) Text() string {
	return goquery.NewDocumentFromNode(e.node).Text()
}

// Link alias of Attr for the first matched of "src/href"
//
//	@return string
func (e *DataElement) Link() string {
	for _, attr := range []string{"src", "href"} {
		if v := e.Attr(attr); v != "" {
			return v
		}
	}

	return ""
}

func (e *DataElement) Target() string {
	t := e.Text()
	l := e.Link()

	if l == "" {
		return t
	}

	return fmt.Sprintf("%s(%s)", t, l)
}

func (e *DataElement) ChildText(selector string) string {
	return strings.TrimSpace(e.DOM.Find(selector).Text())
}

// ChildAttr returns the stripped text content of the first matching
// element's attribute.
func (h *DataElement) ChildAttr(selector, attrName string) string {
	if attr, ok := h.DOM.Find(selector).Attr(attrName); ok {
		return strings.TrimSpace(attr)
	}

	return ""
}
