package main

import (
	"fmt"
	"time"

	"roddy/examples/echoserver"
	"roddy/queue"

	"roddy"

	"github.com/coghost/xlog"
)

func main() {
	xlog.InitLogForConsole()

	go echoserver.Start()

	url := echoserver.ServerURL
	q, _ := queue.New(4, queue.NewInMemory(10000))

	c := roddy.NewCollector(
		roddy.AllowURLRevisit(true),
		roddy.Parallelism(4),
		roddy.RandomDelay(10*time.Second),
	)

	defer c.QuitOnTimeout()

	c.OnRequest(func(r *roddy.Request) {
		fmt.Println("visiting", r.String())
		if r.ID < 15 {
			if r2, err := r.New(fmt.Sprintf("%s?x=%v", url, r.ID)); err == nil {
				q.AddRequest(r2)
			}
		}
	})

	for i := 0; i < 40; i++ {
		q.AddURL(fmt.Sprintf("%s?n=%d", url, i))
	}

	q.Run(c)
}
