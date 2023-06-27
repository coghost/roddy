package main

import (
	"fmt"
	"time"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/coghost/xpretty"
)

func main() {
	xlog.InitLogDebug()
	xpretty.InitializeWithColor()

	c := roddy.NewCollector(
		roddy.Async(true),
		roddy.RandomDelay(1*time.Second),
		roddy.Parallelism(4),
	)

	c.OnHTML("html>body", func(e *roddy.SerpElement) {
		fmt.Println("[from]", e.Request.IDString(), "[got]", e.Text())
	})

	for i := 0; i < 20; i++ {
		c.Visit(fmt.Sprintf("%s?n=%d", "https://postman-echo.com/get", i))
	}

	c.Wait()
}
