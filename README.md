# roddy

## Example

```go
func main() {
	c := roddy.NewCollector()

	// Find and visit all links
	c.OnHTML("a[href]", func(e *roddy.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnRequest(func(r *roddy.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit("http://go-colly.org/")
}
```

See [examples folder](https://github.com/coghost/roddy/tree/main/examples) for more detailed examples.
