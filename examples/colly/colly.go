package main

import (
	"fmt"

	"roddy"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
)

func main() {
	runRoddy()
}

func runRoddy() {
	c := roddy.NewCollector(
		roddy.MaxDepth(2),
	)

	c.OnHTML("a[href]", func(e *roddy.SerpElement) error {
		e.Request.Visit(e.Link())
		return nil
	})

	c.OnRequest(func(r *roddy.Request) {
		fmt.Println(r.String())
	})

	c.Visit("http://go-colly.org/")
}

func runColly() {
	c := colly.NewCollector(
		colly.MaxDepth(2),
		colly.Debugger(&debug.LogDebugger{}),
	)

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit("http://go-colly.org/")
}
