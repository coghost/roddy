package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/go-rod/rod"
)

func main() {
	c := roddy.NewCollector(
		roddy.MaxDepth(2),
	)

	xlog.InitLogForConsole()

	c.OnHTML("head>title", func(e *roddy.SerpElement) error {
		fmt.Printf("%s got title %s\n", e.Request.String(), e.Text())

		return nil
	})

	c.OnHTML(`ul.plainlinks div.wikipedia-languages-langs a[href$='wikipedia.org/wiki/']`,
		func(e *roddy.SerpElement) error {
			link := e.Link()
			fmt.Printf("[From] %s => [Got] %s\n", e.Request.String(), e.Target())
			e.Request.Visit(link)
			return nil
		}, roddy.WithDeferFunc(func(p *rod.Page) {
			c.MustGoBack(p)
		}))

	c.Visit("https://en.wikipedia.org/")
}
