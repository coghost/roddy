package roddy

import "github.com/go-rod/rod"

type Response struct {
	Request *Request
	Page    *rod.Page

	// Ctx is a context between a Request and a Response
	Ctx *Context
}
