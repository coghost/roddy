package main

import (
	"fmt"
	"strings"

	"roddy"

	"github.com/coghost/xbot"
	"github.com/coghost/xdtm"
	"github.com/coghost/xlog"
	"github.com/coghost/xutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/ungerik/go-dry"
)

var (
	_rootDir  = fsutil.ExpandPath("~/tmp/blind/kr_companies")
	_maxPages = 5
)

func save(name, page string, raw string) {
	file := fmt.Sprintf("%s/%s/%s_%s.%s.json", _rootDir, name, name, xdtm.Now().ToShortDateTimeString(), page)
	xutil.MkdirIfNotExistFromFile(file)
	dry.FileSetString(file, raw)
	log.Debug().Str("page", page).Msg("saved")
}

func check(name string) int {
	file := fmt.Sprintf("%s/%s", _rootDir, name)
	arr, _ := dry.ListDirFiles(file)

	return len(arr)
}

func crawl(company string) {
	xlog.InitLog(xlog.WithCaller(true), xlog.WithLevel(zerolog.DebugLevel), xlog.WithNoColor(false), xlog.WithTimestampFunc(xlog.LocalFn))
	log.Debug().Str("company", company).Msg("working on")

	crawledPages := check(company)
	if crawledPages >= _maxPages {
		log.Debug().Str("company", company).Int("pages", crawledPages).Msg("found")
		return
	}

	const (
		sortByDate = "div.sorting>button+button"
		postScript = `() => { return JSON.stringify(document.querySelector('div.ctbox>div').__vue__.posts) }`
		paginate   = `div.paginate>strong~a[class=""],div.paginate>strong~a[class="nuxt-link-active"]`
	)

	c := roddy.NewCollector(
		roddy.IgnoreVistedError(true),
		roddy.MaxPageNum(uint32(_maxPages)),
	)
	defer c.ClearBot()

	c.OnHTML(sortByDate, func(e *roddy.SerpElement) error {
		log.Debug().Msg("toggle sort by date option")
		err := e.Click(e.Selector)
		c.OnHTMLDetach(sortByDate)
		return err
	})

	c.OnPaging(paginate, func(e *roddy.SerpElement) error {
		pg := cast.ToInt(e.Bot.GetElementAttr(`div.paginate>strong`))

		active := e.Bot.GetElementAttr(sortByDate, xbot.ElemAttr("class"))
		if active != "on" {
			return fmt.Errorf("sort by date is not activated.")
		}

		rawJson := e.Bot.Pg.MustEval(postScript).String()
		save(company, cast.ToString(pg), rawJson)

		// by clicking the largest page num, can speed up the crawler.
		index := 0
		if crawledPages-pg > 1 {
			index = -1
		}
		index = 0

		c.UpdatePageNum(uint32(pg))
		e.ClickAtIndex(e.Selector, index)
		return e.Request.VisitByMockClick()
	})

	uri := fmt.Sprintf("https://www.teamblind.com/kr/company/%s/posts/%s", company, company)

	c.Visit(uri)
}

func main() {
	lines, err := dry.FileGetLines("/tmp/keywords.txt")
	if err != nil {
		log.Error().Err(err).Msg("cannot read file")
		return
	}

	for _, comp := range lines {
		if strings.TrimSpace(comp) == "" {
			continue
		}

		crawl(comp)
	}
}
