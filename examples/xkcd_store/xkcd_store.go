package main

import (
	"encoding/csv"
	"os"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/rs/zerolog/log"
)

func main() {
	xlog.InitLog()

	fName := "/tmp/cryptocoin_market.csv"

	file, err := os.Create(fName)
	if err != nil {
		log.Fatal().Msg("cannot create file")
	}

	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	writer.Write([]string{"Name", "Price", "URL", "Image URL"})

	c := roddy.NewCollector(
		roddy.AllowedDomains("store.xkcd.com"),
	)

	c.OnData(`.product-grid-item`, func(e *roddy.DataElement) {
		writer.Write([]string{
			e.ChildAttr("a", "title"),
			e.ChildText("span"),
			e.ChildAttr("a", "href"),
			e.ChildAttr("img", "src"),
		})
	})

	c.OnPaging(`.next a[href]`, func(e *roddy.SerpElement) {
		e.Request.Visit(e.Link())
	})

	c.Visit("https://store.xkcd.com/collections/everything")
}
