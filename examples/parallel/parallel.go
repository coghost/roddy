package main

import (
	"time"

	"roddy"

	"github.com/coghost/xlog"
)

func main() {
	xlog.InitLogDebug()

	c := roddy.NewCollector(
		roddy.MaxDepth(2),
		roddy.Async(true),
		roddy.Parallelism(4),
		roddy.RandomDelay(1*time.Second),
	)

	c.OnHTML("a[href$='wikipedia.org/wiki/']", func(e *roddy.SerpElement) {
		link := e.Link()
		e.Request.Visit(link)
	})

	c.Visit("https://en.wikipedia.org/")

	c.Wait()
}
