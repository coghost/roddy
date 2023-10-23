package roddy

import (
	"fmt"

	"github.com/coghost/xbot"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/k0kubun/pp/v3"
)

type SerpElement struct {
	Selector string

	DOM *rod.Element

	Bot      *xbot.Bot
	Request  *Request
	Response *Response

	root *rod.Element

	Index int
}

func NewSerpElement(resp *Response, elem *rod.Element, name string, index int) *SerpElement {
	return &SerpElement{
		Selector: name,
		DOM:      elem,
		Request:  resp.Request,
		Response: resp,
		Index:    index,

		Bot: xbot.NewBotWithPage(resp.Page),
	}
}

func (e *SerpElement) MarkElemAsRoot() {
	e.root = e.DOM
}

func (e *SerpElement) ResetRoot() {
	e.root = e.Bot.Pg.MustElement("html")
}

func (e *SerpElement) Attr(k string) string {
	v, err := e.DOM.Attribute(k)
	if err != nil || v == nil {
		return ""
	}

	return *v
}

func (e *SerpElement) Text() string {
	v, err := e.DOM.Text()
	if err != nil {
		return ""
	}

	return v
}

// Link alias of Attr for the first matched of "src/href"
//
//	@return string
func (e *SerpElement) Link() string {
	for _, attr := range []string{"src", "href"} {
		if v := e.Attr(attr); v != "" {
			return v
		}
	}

	return ""
}

func (e *SerpElement) Target() string {
	t := e.Text()
	l := e.Link()

	if l == "" {
		return t
	}

	return fmt.Sprintf("%s(%s)", t, l)
}

func (e *SerpElement) UpdateText(selector string, text string) (string, error) {
	return e.Bot.FillBar(selector, text, xbot.WithRoot(e.root))
}

func (e *SerpElement) Click(selector string) error {
	e.Bot.GetElem(selector, xbot.WithRoot(e.root))

	err := e.Bot.ScrollAndClick(selector, xbot.WithRoot(e.root))
	if err != nil {
		pp.Println(err)
	}

	return err
}

func (e *SerpElement) ClickAtIndex(selector string, index int) error {
	elem := e.Bot.GetElem(selector, xbot.WithRoot(e.root), xbot.ElemIndex(index))

	err := e.Bot.ClickElem(elem)
	if err != nil {
		pp.Println(err)
	}

	return err
}

// ScrollUntilElemInteractable
func (e *SerpElement) ScrollUntilElemInteractable(selector string, maxStep int) {
	for i := 0; i < maxStep; i++ {
		e.DOM.MustKeyActions().Press(input.PageDown).MustDo()

		more := e.Bot.GetElem(selector)

		if _, err := more.Interactable(); err == nil {
			break
		}
	}
}

func (e *SerpElement) Focus(count int, style string) {
	if count <= 0 {
		return
	}

	e.Bot.ScrollToElemDirectly(e.DOM)
	e.Bot.HighlightBlink(e.DOM, count, style)
}
