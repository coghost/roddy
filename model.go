package roddy

import (
	"github.com/go-rod/rod"
)

// RequestCallback is a type alias for OnRequest callback functions
type RequestCallback func(*Request)

// ResponseCallback is a type alias for OnResponse callback functions
type ResponseCallback func(*Response)

// ErrorCallback is a type alias for OnError callback functions
type ErrorCallback func(*Response, error)

// HTMLCallback is a type alias for OnHTML callback functions
type HTMLCallback func(e *SerpElement) error

// DataCallback is where we handle all data wanted
type DataCallback func(e *DataElement)

// ScrapedCallback is a type alias for OnScraped callback functions
type ScrapedCallback func(*Response)

type htmlCallbackContainer struct {
	Selector string
	Function HTMLCallback

	DeferFunc func(p *rod.Page)
}

type dataCallbackContainer struct {
	Selector string
	Function DataCallback
}
