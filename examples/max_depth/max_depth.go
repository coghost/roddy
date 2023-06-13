package main

import (
	"fmt"

	"roddy"
)

func main() {
	c := roddy.NewCollector(
		roddy.MaxDepth(2),
	)
	// defer c.HangUpHourly()

	c.OnHTML("a[href$='wikipedia.org/wiki/']", func(e *roddy.HTMLElement) {
		link := e.Attr("href")
		fmt.Printf("visiting [%s](%s)\n", e.Text, link)
		// Visit link found on page
		e.Request.Visit(link)
		// err := e.Request.Visit(link)
		// if err != nil {
		// 	fmt.Println("got error:", e.Request.String(), err)
		// }
	})

	c.Visit("https://en.wikipedia.org/")
}
