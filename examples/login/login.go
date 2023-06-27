package main

import (
	"fmt"
	"sync"

	"roddy"

	"github.com/coghost/xlog"
)

func login2scrape() {
	c := roddy.NewCollector()
	// defer c.QuitOnTimeout()

	xlog.InitLogForConsole()

	c.OnHTML("form.el-form", func(e *roddy.SerpElement) {
		e.MarkElemAsRoot()

		e.UpdateText(`input[type="text"]`, "Admin")
		e.UpdateText(`input[type="password"]`, "123456")
		e.Click(`button[type="button"]`)
	})

	c.OnHTML(`a[href="/"]`, func(e *roddy.SerpElement) {
		e.ResetRoot()

		cls := e.Attr("class")
		fmt.Printf("got class: %q\n", cls)
	})

	c.Visit("https://login3.scrape.center/login")
}

func login2spiderbuf() {
	c := roddy.NewCollector()
	defer c.QuitOnTimeout()

	xlog.InitLogForConsole(xlog.WithLevel(1))

	c.OnHTML("form.form-horizontal", func(e *roddy.SerpElement) {
		e.MarkElemAsRoot()

		e.UpdateText(`input#username`, "admin")
		e.UpdateText(`input#password`, "123456")
		e.Click(`button.btn`)

		e.ResetRoot()
	})

	c.OnHTML("table.table>tbody>tr", func(e *roddy.SerpElement) {
		fmt.Println(e.Text())
	})

	c.Visit("http://www.spiderbuf.cn/e01/")
}

func runAsync() {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		login2scrape()
	}()

	go func() {
		defer wg.Done()
		login2spiderbuf()
	}()

	wg.Wait()
}

func main() {
	runAsync()
}
