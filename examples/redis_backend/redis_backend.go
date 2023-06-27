package main

import (
	"fmt"
	"os"

	"roddy"
	"roddy/queue"

	"github.com/coghost/xlog"
	"github.com/gocolly/redisstorage"
	"github.com/k0kubun/pp/v3"
	"github.com/spf13/cast"
)

func main() {
	xlog.InitLogDebug()

	addr, db, pwd := parseArgs()
	fmt.Printf("connecting to %s<%d> with pwd:(%s)\n", addr, db, pwd)

	_cap := 2

	c := roddy.NewCollector(
		roddy.Parallelism(_cap),
	)

	storage := &redisstorage.Storage{
		Address:  addr,
		Password: pwd,
		DB:       db,
		Prefix:   "httpbin_test",
	}

	if err := c.SetStorage(storage); err != nil {
		panic(err)
	}

	if err := storage.Clear(); err != nil {
		panic(err)
	}

	defer storage.Client.Close()

	q, _ := queue.New(_cap, storage)

	c.OnResponse(func(r *roddy.Response) {
		pp.Println(r.Request.IDString(), r.Page.MustCookies())
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

func parseArgs() (string, int, string) {
	addr := "127.0.0.1:6379"
	db := 0
	pwd := ""

	args := os.Args[1:]
	switch len(args) {
	case 0:
		break
	case 1:
		addr = args[0]
		fallthrough
	case 2:
		addr = args[0]
		db = cast.ToInt(args[1])
	case 3:
		addr = args[0]
		db = cast.ToInt(args[1])
		pwd = args[2]
	default:
		fmt.Println("USAGE: program <ADDR> <DB> <PWD>")
		os.Exit(0)
	}

	return addr, db, pwd
}
