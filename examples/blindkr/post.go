package main

import (
	"fmt"

	"github.com/coghost/xbot"
	"github.com/coghost/xlog"
	"github.com/coghost/xutil"
	"github.com/gookit/goutil/dump"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func __main() {
	xlog.InitLog(xlog.WithCaller(true), xlog.WithLevel(zerolog.DebugLevel), xlog.WithNoColor(false), xlog.WithTimestampFunc(xlog.LocalFn))

	bot := xbot.NewDefaultBot(true)

	company := "테이스티나인" // 940
	// company = "간호사" // na
	company = "모나미" // 394
	uri := fmt.Sprintf("https://www.teamblind.com/kr/company/%s/posts/%s", company, company)
	log.Debug().Str("company", company).Msg("working on")

	// get page
	bot.GetPage(uri)

	// if no company found, wait for manual input
	// click sort by date, and wait ready

	na := "section.not-found"
	sortByDate := "div.sorting>button+button"

	got := bot.MustEnsureAnyElem(na, sortByDate)
	if got == na {
		xutil.Pause("something wrong happens, enter the company manually to continue")
	}

	bot.ScrollAndClick(sortByDate)
	xutil.RandSleep(0.5, 1.0)

	for i := 0; i < 5; i++ {
		dump.P("saving current page")

		btn := bot.GetElem(`div.paginate>strong+a:not(.nuxt-link-exact-active)`)
		if btn == nil {
			break
		}

		bot.ScrollToElem(btn)
		bot.ClickElem(btn)
		// wait new page loaded
	}

	// check if has next page
	// scroll to bottom, and click next page
}
