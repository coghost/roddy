package main

import (
	"fmt"

	"roddy/queue"

	"roddy"

	"github.com/coghost/xlog"
)

func main() {
	xlog.InitLogForConsole()

	url := "https://httpbin.org/delay/1"
	q, _ := queue.New(2, queue.NewInMemory(10000))
	c := roddy.NewCollector(roddy.QuitInSeconds(3), roddy.AllowURLRevisit(true))

	c.OnRequest(func(r *roddy.Request) {
		fmt.Println("visiting", r.String())
		if r.ID < 15 {
			if r2, err := r.New(fmt.Sprintf("%s?x=%v", url, r.ID)); err == nil {
				q.AddRequest(r2)
			}
		}
	})

	for i := 0; i < 5; i++ {
		q.AddURL(fmt.Sprintf("%s?n=%d", url, i))
	}

	q.Run(c)
}
