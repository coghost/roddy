package main

import (
	"encoding/csv"
	"fmt"
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
	writer.Write([]string{"ID", "Name", "Symbol", "Price (USD)", "Volume (USD)", "Market capacity (USD)", "Change (1h)", "Change (24h)", "Change (7d)"})

	c := roddy.NewCollector()
	home := "https://coinmarketcap.com/all/views/all/"

	c.OnHTML(`html>body.DAY`, func(e *roddy.SerpElement) error {
		// please beware of `html>body.DAY`, this selector finds and only finds one element.
		// by default, only partial records are loaded, so scroll until load more is clickable.
		e.ScrollUntilElemInteractable(`div.cmc-table-listing__loadmore>button`, 64)
		return nil
	})

	id := 0

	c.OnData("tbody tr", func(e *roddy.DataElement) {
		id++
		writer.Write([]string{
			fmt.Sprintf("%d", id),
			e.ChildText("div.cmc-table__column-name"),
			e.ChildText(".cmc-table__cell--sort-by__symbol"),
			e.ChildText(".cmc-table__cell--sort-by__market-cap"),
			e.ChildText(".cmc-table__cell--sort-by__price"),
			e.ChildText(".cmc-table__cell--sort-by__circulating-supply"),
			e.ChildText(".cmc-table__cell--sort-by__volume-24-h"),
			e.ChildText(".cmc-table__cell--sort-by__percent-change-1-h"),
			e.ChildText(".cmc-table__cell--sort-by__percent-change-24-h"),
			e.ChildText(".cmc-table__cell--sort-by__percent-change-7-d"),
		})
	})

	c.Visit(home)
}
