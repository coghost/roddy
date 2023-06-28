package main

import (
	"fmt"
	"sync"

	"roddy"

	"github.com/coghost/xlog"
)

func login2scrape() {
	c := roddy.NewCollector()
	defer c.QuitOnTimeout()

	xlog.InitLogForConsole()

	c.OnHTML("form.el-form", func(e *roddy.SerpElement) {
		e.MarkElemAsRoot()

		e.UpdateText(`input[type="text"]`, "admin")
		e.UpdateText(`input[type="password"]`, "admin")
		e.Click(`button[type="button"]`)
	})

	c.OnHTML(`div.el-message--success`, func(e *roddy.SerpElement) {
		fmt.Println(e.Text())
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

func runAsync(args ...func()) {
	wg := sync.WaitGroup{}
	for _, fn := range args {
		wg.Add(1)

		go func(fn func()) {
			defer wg.Done()
			fn()
		}(fn)
	}

	wg.Wait()
}

func main() {
	runAsync(login2scrape)
}
