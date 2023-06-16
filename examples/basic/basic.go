package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/coghost/xpretty"
)

func main() {
	c := roddy.NewCollector(
		roddy.AllowedDomains("hackerspaces.org", "wiki.hackerspaces.org"),
	)

	xlog.InitLogForConsole()

	c.OnHTML("a[href]", func(e *roddy.HTMLElement) {
		link := e.Attr("href")
		fmt.Printf("from %s, found: %q -> %s\n", e.Request.String(), e.Text(), link)

		c.Visit(e.Request.AbsoluteURL(link))
	})

	c.OnRequest(func(r *roddy.Request) {
		xpretty.YellowPrintf("Visiting %s\n", r.String())
	})

	c.Visit("https://hackerspaces.org/")
}
