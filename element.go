package roddy

import (
	"fmt"

	"github.com/coghost/xbot"
	"github.com/go-rod/rod"
)

// HTMLCallback is a type alias for OnHTML callback functions
type HTMLCallback func(e *HTMLElement)

type htmlCallbackContainer struct {
	Selector string
	Function HTMLCallback

	DeferFunc func()
}

type HTMLElement struct {
	Selector string

	DOM *rod.Element

	Bot      *xbot.Bot
	Request  *Request
	Response *Response

	Index int
}

func NewHTMLElement(resp *Response, elem *rod.Element, name string, index int) *HTMLElement {
	return &HTMLElement{
		Selector: name,
		DOM:      elem,
		Request:  resp.Request,
		Response: resp,
		Index:    index,

		Bot: resp.Request.collector.Bot,
	}
}

func (e *HTMLElement) Attr(k string) string {
	v, err := e.DOM.Attribute(k)
	if err != nil || v == nil {
		return ""
	}

	return *v
}

func (e *HTMLElement) Text() string {
	v, err := e.DOM.Text()
	if err != nil {
		return ""
	}

	return v
}

// Link alias of Attr for the first matched of "src/href"
//
//	@return string
func (e *HTMLElement) Link() string {
	for _, attr := range []string{"src", "href"} {
		if v := e.Attr(attr); v != "" {
			return v
		}
	}

	return ""
}

func (e *HTMLElement) Target() string {
	t := e.Text()
	l := e.Link()

	if l == "" {
		return t
	}

	return fmt.Sprintf("%s(%s)", t, l)
}

func (e *HTMLElement) UpdateText(selector string, text string) (string, error) {
	return e.Bot.FillBar(selector, text)
}

func (e *HTMLElement) Click(selector string) error {
	return e.Bot.ScrollAndClick(selector)
}

func (e *HTMLElement) Focus(count int, style string) {
	if count <= 0 {
		return
	}

	e.Bot.ScrollToElemDirectly(e.DOM)
	e.Bot.HighlightBlink(e.DOM, count, style)
}
