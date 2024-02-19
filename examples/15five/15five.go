package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/gocolly/colly"

	"github.com/go-rod/rod"
)

const (
	home = "https://www.15five.com/"
)

func main() {
	runColly()
}

func runRoddy() {
	xlog.InitLogDebug()

	c := roddy.NewCollector(
		roddy.MaxDepth(2),
	)

	c.OnHTML(`a[href]`, func(e *roddy.SerpElement) error {
		link := e.Link()
		fmt.Printf("%s", e)
		e.Request.Visit(link)
		return nil
	}, roddy.WithDeferFunc(func(p *rod.Page) {
		c.MustGoBack(p)
	}))

	c.Visit(home)
}

func runColly() {
	c := colly.NewCollector(
		colly.MaxDepth(2),
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit(home)
}
