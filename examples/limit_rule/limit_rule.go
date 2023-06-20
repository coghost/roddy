package main

import (
	"fmt"
	"time"

	"roddy"
	"roddy/examples/echoserver"

	"github.com/coghost/xlog"
	"github.com/coghost/xpretty"
)

func main() {
	go echoserver.Start()

	xlog.InitLogDebug(xlog.WithLevel(0))
	xpretty.InitializeWithColor()

	c := roddy.NewCollector(
		roddy.Async(true),
		roddy.HighlightCount(4),
		roddy.RandomDelay(1*time.Second),
		roddy.Parallelism(2),
	)

	c.OnHTML("html>body", func(e *roddy.HTMLElement) {
		fmt.Println("[from]", e.Request.IDString(), "[got]", e.Text()[:32])
	})

	for i := 0; i < 10; i++ {
		c.Visit(fmt.Sprintf("%s?n=%d", echoserver.ServerURL, i))
	}

	c.Wait()
}
