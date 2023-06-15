package main

import (
	"fmt"

	"roddy"
)

func main() {
	c := roddy.NewCollector()
	defer c.HangUpInSeconds()

	c.OnRequest(func(r *roddy.Request) {
		r.Ctx.Put("url", r.String())
	})

	c.OnResponse(func(r *roddy.Response) {
		fmt.Println(r.Ctx.Get("url"))
	})

	c.Visit("https://en.wikipedia.org/")
}
