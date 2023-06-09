package main

import (
	"fmt"

	"roddy"
)

func main() {
	c := roddy.NewCollector()

	c.OnHTML("*", func(e *roddy.HTMLElement) {
		fmt.Println(e)
	})

	c.OnError(func(r *roddy.Response, err error) {
		fmt.Println("Request URL:", r.Request.String(), "failed with response:", r, "\nError:", err)
	})

	c.Visit("https://definitely-not-a.website/")
}
