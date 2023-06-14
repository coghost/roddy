package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xpretty"
)

func main() {
	c := roddy.NewCollector(
		roddy.AllowedDomains("hackerspaces.org", "wiki.hackerspaces.org"),
		roddy.Headless(false),
	)

	c.OnHTML("a[href]", func(e *roddy.HTMLElement) {
		link := e.Attr("href")
		fmt.Printf("Link found: %q -> %s\n", e.Text(), link)

		c.Visit(e.Request.AbsoluteURL(link))
	})

	c.OnRequest(func(r *roddy.Request) {
		xpretty.YellowPrintf("Visiting %s\n", r.String())
	})

	c.Visit("https://hackerspaces.org/")
}
