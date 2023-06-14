package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xlog"
)

func main() {
	c := roddy.NewCollector(
		roddy.MaxDepth(2),
		roddy.IgnoredErrors(roddy.ErrMaxDepth),
		roddy.IgnoreVistedError(true),
	)

	xlog.InitLogForConsole()

	c.OnHTML("a[href$='wikipedia.org/wiki/']", func(e *roddy.HTMLElement) {
		link := e.Attr("href")
		if e.Text() == "" {
			return
		}
		fmt.Printf("[From] %s => [Got] %s:%s\n", e.Request.String(), e.Text(), link)
		e.Request.Visit(link)
	})

	c.Visit("https://en.wikipedia.org/")
}
