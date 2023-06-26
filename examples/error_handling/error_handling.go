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

	c.OnHTML("*", func(e *roddy.HTMLElement) {
		fmt.Println(e)
	})

	c.OnError(func(r *roddy.Response, err error) {
		fmt.Println("Request URL:", r.Request.String(), "failed with response:", r, "\nError:", err)
	})

	c.Visit("https://definitely-not-a.website/")
}
