package roddy

import (
	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
)

// HTMLCallback is a type alias for OnHTML callback functions
type HTMLCallback func(elem *HTMLElement)

type htmlCallbackContainer struct {
	Selector string
	Function HTMLCallback
}

type HTMLElement struct {
	Selector string

	Elem *rod.Element

	Request  *Request
	Response *Response

	Index int
}

func NewHTMLElement(resp *Response, elem *rod.Element, name string, index int) *HTMLElement {
	return &HTMLElement{
		Selector: name,
		Elem:     elem,
		Request:  resp.Request,
		Response: resp,
		Index:    index,
	}
}

func (e *HTMLElement) Attr(k string) string {
	v, err := e.Elem.Attribute(k)
	if err != nil {
		log.Error().Err(err).Str("attr", k).Msg("cannot get attr")
		return ""
	}
	return *v
}

func (e *HTMLElement) Text() string {
	v, err := e.Elem.Text()
	if err != nil {
		log.Error().Err(err).Str("selector", e.Selector).Msg("cannot get text")
		return ""
	}
	return v
}
