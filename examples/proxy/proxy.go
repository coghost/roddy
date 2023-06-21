package main

import (
	"fmt"
	"math/rand"
	"time"

	"roddy"

	"github.com/coghost/xlog"
)

func main() {
	// Free Proxy List: https://www.freeproxylists.net/?s=u
	c := roddy.NewCollector(
		roddy.AllowURLRevisit(true),
		roddy.WithProxies(
			"171.227.1.137:10066",
			"58.246.58.150:9002",
			"120.197.40.219:9002",
		),
	)
	defer c.QuitOnTimeout(1)

	rand.New(rand.NewSource(time.Now().Unix()))

	xlog.InitLogForConsole()

	c.OnHTML("body", func(e *roddy.HTMLElement) {
		fmt.Println(e.Text())
	})

	c.Visit("http://ip.42.pl/short")
}
