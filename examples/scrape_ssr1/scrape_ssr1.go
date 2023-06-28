package main

import (
	"fmt"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/gookit/goutil/dump"
)

func main() {
	xlog.InitLogDebug()

	c := roddy.NewCollector()

	c.OnData(`div.item`, func(e *roddy.DataElement) {
		fmt.Printf(
			"%s(%s) %s\n\t%s\n",
			e.ChildText("a>h2"),
			e.ChildText("p.score"),
			e.ChildText("div.categories+div.info+div.info"),
			e.ChildAttr("a>img", "src"),
		)
	})

	c.OnPaging(`a.next`, func(e *roddy.SerpElement) {
		e.Bot.GetElem(`div.item`)
		e.Click(e.Selector)
		e.Request.Visit(roddy.BlankPagePlaceholder)
	})

	c.OnError(func(r *roddy.Response, err error) {
		dump.P(err)
	})

	c.Visit("https://ssr1.scrape.center/")
}
