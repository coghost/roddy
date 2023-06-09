package roddy

import (
	"github.com/go-rod/rod"
)

type Response struct {
	Request *Request
	Page    *rod.Page

	// Ctx is a context between a Request and a Response
	Ctx *Context
}

// ResponseCallback is a type alias for OnResponse callback functions
type ResponseCallback func(*Response)

// ErrorCallback is a type alias for OnError callback functions
type ErrorCallback func(*Response, error)
