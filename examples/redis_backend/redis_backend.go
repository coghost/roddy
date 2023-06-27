package main

import (
	"roddy"
	"roddy/queue"

	"github.com/coghost/xlog"
	"github.com/gocolly/redisstorage"
	"github.com/k0kubun/pp/v3"
)

func main() {
	xlog.InitLogDebug()

	c := roddy.NewCollector()

	storage := &redisstorage.Storage{
		Address:  "127.0.0.1:6379",
		Password: "",
		DB:       0,
		Prefix:   "httpbin_test",
	}

	if err := c.SetStorage(storage); err != nil {
		panic(err)
	}

	if err := storage.Clear(); err != nil {
		panic(err)
	}

	defer storage.Client.Close()

	q, _ := queue.New(2, storage)

	c.OnResponse(func(r *roddy.Response) {
		pp.Println(r.Page.MustCookies())
	})

	urls := []string{
		"http://httpbin.org/",
		"http://httpbin.org/ip",
		"http://httpbin.org/cookies/set?a=b&c=d",
		"http://httpbin.org/cookies",
	}
	for _, u := range urls {
		q.AddURL(u)
	}

	q.Run(c)
}
