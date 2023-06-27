package main

import (
	"encoding/json"
	"net/http"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/k0kubun/pp/v3"
	"github.com/rs/zerolog/log"
)

type pageInfo struct {
	Links map[string]int
	Total int
	Page  string
}

func handler(w http.ResponseWriter, r *http.Request) {
	URL := r.URL.Query().Get("url")
	if URL == "" {
		log.Warn().Msg("missing URL argument")
		return
	}

	log.Info().Str("url", URL).Msg("visiting")

	c := roddy.NewCollector()
	defer c.QuitOnTimeout(1)

	p := &pageInfo{Links: make(map[string]int)}

	c.OnHTML("a[href]", func(e *roddy.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Link())
		if link != "" {
			p.Links[link]++
			p.Total++
		}
	})

	c.OnResponse(func(r *roddy.Response) {
		p.Page = r.Page.String()
	})

	c.OnError(func(r *roddy.Response, err error) {
		p.Page = r.Page.String() + err.Error()
	})

	c.Visit(URL)

	b, err := json.Marshal(p)
	if err != nil {
		log.Error().Err(err).Msg("cannot serialize response")
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

func main() {
	xlog.InitLogForConsole(xlog.WithLevel(0))

	addr := ":7171"

	pp.Println("USAGE: http://127.0.0.1:7171/?url=http://go-colly.org/")

	http.HandleFunc("/", handler)
	log.Info().Str("addr", addr).Msg("listening on")
	log.Fatal().Err(http.ListenAndServe(addr, nil)).Msg("cannot listen and serve")
}
