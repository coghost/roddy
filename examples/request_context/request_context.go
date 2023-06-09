package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xlog"
)

func main() {
	xlog.InitLogDebug()

	c := roddy.NewCollector()
	defer c.QuitOnTimeout()

	c.OnRequest(func(r *roddy.Request) {
		r.Ctx.Put("url", r.String())
	})

	c.OnResponse(func(r *roddy.Response) {
		fmt.Println(r.Ctx.Get("url"))
	})

	c.Visit("https://en.wikipedia.org/")
}
