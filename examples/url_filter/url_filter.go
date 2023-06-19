package main

import (
	"fmt"
	"regexp"

	"roddy"

	"github.com/coghost/xlog"
)

func main() {
	c := roddy.NewCollector(
		roddy.URLFilters(
			regexp.MustCompile("http://httpbin\\.org/(|e.+)$"),
			regexp.MustCompile("https://github\\.com/req.+"),
		),
		roddy.QuitInSeconds(),
	)

	xlog.InitLogForConsole()

	c.OnHTML("a[href]", func(e *roddy.HTMLElement) {
		link := e.Attr("href")
		fmt.Printf("Link found: %q -> %s\n", e.Text(), link)

		c.Visit(e.Request.AbsoluteURL(link))
	})

	c.OnRequest(func(r *roddy.Request) {
		fmt.Println("Visiting", r.String())
	})

	c.Visit("http://httpbin.org/")
}
