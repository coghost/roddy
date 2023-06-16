package roddy

import "github.com/go-rod/rod"

type SerpCallback func(e *SerpElement)

type serpCallbackContainer struct {
	Selector string
	Function SerpCallback
}

type SerpElement struct {
	*HTMLElement
}

func NewSerpElement(resp *Response, elem *rod.Element, name string, index int) *SerpElement {
	return &SerpElement{
		NewHTMLElement(resp, elem, name, index),
	}
}
