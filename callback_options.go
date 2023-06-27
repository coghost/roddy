package roddy

import "github.com/go-rod/rod"

type CallbackOptions struct {
	deferFunc func(p *rod.Page)
}

type CallbackOptionFunc func(o *CallbackOptions)

func bindCallbackOptions(opt *CallbackOptions, opts ...CallbackOptionFunc) {
	for _, f := range opts {
		f(opt)
	}
}

func WithDeferFunc(fn func(p *rod.Page)) CallbackOptionFunc {
	return func(o *CallbackOptions) {
		o.deferFunc = fn
	}
}
