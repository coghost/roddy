package roddy

import "github.com/go-rod/rod"

type OnHTMLDeferFn func(p *rod.Page)

type OnHTMLOptions struct {
	deferFunc OnHTMLDeferFn
}

type OnHTMLOptionFunc func(o *OnHTMLOptions)

func bindOnHTMLOptions(opt *OnHTMLOptions, opts ...OnHTMLOptionFunc) {
	for _, f := range opts {
		f(opt)
	}
}

func WithDeferFunc(fn OnHTMLDeferFn) OnHTMLOptionFunc {
	return func(o *OnHTMLOptions) {
		o.deferFunc = fn
	}
}
