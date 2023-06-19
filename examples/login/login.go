package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xlog"
)

func login2scrape() {
	c := roddy.NewCollector(roddy.QuitInSeconds())

	xlog.InitLogForConsole()

	c.OnSerp("form.el-form", func(e *roddy.SerpElement) {
		e.UpdateText(`input[type="text"]`, "Admin")
		e.UpdateText(`input[type="password"]`, "123456")
		e.Click(`button[type="button"]`)
	})

	c.OnHTML(`a[href="/"]`, func(e *roddy.HTMLElement) {
		cls := e.Attr("class")
		fmt.Printf("got class: %q\n", cls)
	})

	c.Visit("https://login3.scrape.center/login")
}

func login2spiderbuf() {
	c := roddy.NewCollector(roddy.QuitInSeconds())

	xlog.InitLogForConsole(xlog.WithLevel(1))

	c.OnSerp("form.form-horizontal", func(e *roddy.SerpElement) {
		e.UpdateText(`input#username`, "admin")
		e.UpdateText(`input#password`, "123456")
		e.Click(`button.btn`)
	})

	c.OnHTML("table.table>tbody>tr", func(e *roddy.HTMLElement) {
		fmt.Println(e.Text())
	})

	c.Visit("http://www.spiderbuf.cn/e01/")
}

func main() {
	login2scrape()
	login2spiderbuf()
}
