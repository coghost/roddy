package roddy

import (
	"net/url"
	"path"
	"time"

	"github.com/coghost/xbot"
	"github.com/coghost/xutil"
	"github.com/go-rod/rod"
	"github.com/gookit/goutil/arrutil"
	"github.com/rs/zerolog/log"
)

/*
overview level:

*/

const (
	_leftStep = 100
	_topStep  = 80
)

var (
	_left = 0
	_top  = 0
)

func (c *Collector) MustGet(request *Request, page *rod.Page, URL *url.URL, depth int) (*Response, error) {
	if URL != nil {
		log.Debug().Str("request", request.String()).Msg("visiting")
		if err := page.Timeout(xbot.MediumToSec * time.Second).Navigate(URL.String()); err != nil {
			log.Error().Err(err).Str("url", URL.String()).Msg("cannot visit")
			return nil, err
		}
	}

	err := page.Timeout(xbot.MediumToSec * time.Second).WaitLoad()
	if err != nil {
		return nil, err
	}

	if URL == nil {
		URL, err = c.getParsedURL(page.MustInfo().URL, depth)
		if err != nil {
			return nil, err
		}

		request.URL = URL
	}

	return &Response{
		Request: request,
		Page:    page,
	}, nil
}

func (c *Collector) MustGoBack(page *rod.Page) {
	if c.maxDepth == 0 {
		return
	}

	err := page.NavigateBack()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot go back")
	}

	err = page.WaitLoad()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot wait load")
	}
}

// func (c *Collector) DumpCookies() error {
// 	ck, err := c.getCookieName(c.Bot.CurrentUrl())
// 	if err != nil {
// 		return err
// 	}

// 	return dry.FileSetJSON(ck, c.Bot.Pg.MustCookies())
// }

// func (c *Collector) SetCookies(url string, page *rod.Page) error {
// 	ck, err := c.getCookieName(url)
// 	if err != nil {
// 		return err
// 	}

// 	raw, err := dry.FileGetString(ck)
// 	if err != nil {
// 		return err
// 	}

// 	return c.Bot.SetPageWithCookies(page, raw)
// }

func (c *Collector) initBotPagePool() {
	c.parallelism = xutil.AorB(c.parallelism, 1)
	c.botPool = NewBotPoolManager(c.parallelism)
	c.pagePool = rod.NewPagePool(c.parallelism)

	c.waitChan = make(chan bool, c.parallelism)
}

func (c *Collector) ClearBot() {
	bot := c.botPool.Get(func() *xbot.Bot {
		bot := c.newBot()
		xbot.SpawnBrowserOnly(bot)
		return bot
	})

	bot.Close()
}

func (c *Collector) createPage() (*xbot.Bot, *rod.Page) {
	log.Trace().Msg("try get page")

	bot := c.createBot()
	defer c.botPool.Put(bot)

	page := c.pagePool.Get(func() *rod.Page {
		page := xbot.CustomizePage(bot.Brw, xbot.Incognito(true))
		return page
	})
	defer c.pagePool.Put(page)

	log.Trace().Str("page", page.String()).Msg("got page")

	return bot, page
}

func (c *Collector) createBot() *xbot.Bot {
	log.Trace().Msg("try get bot")

	bot := c.botPool.Get(func() *xbot.Bot {
		bot := c.newBot()
		xbot.SpawnBrowserOnly(bot)
		return bot
	})

	log.Trace().Str("botId", bot.UniqueID).Msg("got bot")

	return bot
}

func (c *Collector) newBot() *xbot.Bot {
	proxy := ""

	if len(c.proxies) != 0 {
		proxy = arrutil.RandomOne(c.proxies)
	}

	bc := xbot.NewDefaultBotCfg()
	bc.UserAgent = xbot.UA

	bof := []xbot.BotOptFunc{
		xbot.BotSpawn(false),
		xbot.BotScreenPosition(_left, _top),
		xbot.BotHeadless(c.headless),
		xbot.BotUserAgent(c.userAgent),
		xbot.BotProxyServer(proxy),
		xbot.WithBotConfig(bc),
		xbot.BotHighlight(false),
	}

	_left += _leftStep
	_top += _topStep

	return xbot.NewBot(bof...)
}

func (c *Collector) getCookieName(url string) (string, error) {
	name, err := FilenameFromUrl(url)
	if err != nil {
		return "", err
	}

	return path.Join(c.cookieDir, name) + ".cookie.json", nil
}
