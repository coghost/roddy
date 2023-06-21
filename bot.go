package roddy

import (
	"path"

	"github.com/coghost/xbot"
	"github.com/coghost/xutil"
	"github.com/go-rod/rod"
	"github.com/gookit/goutil/arrutil"
	"github.com/rs/zerolog/log"
)

func (c *Collector) MustGoBack(args ...*rod.Page) {
	if c.maxDepth == 0 {
		return
	}

	var page *rod.Page
	if len(args) > 0 {
		page = args[1]
	} else {
		panic("page is required")
	}

	err := page.NavigateBack()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot go back")
	}

	page.MustWaitLoad()
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
	c.botPool = NewBotPool(c.parallelism)
	c.pagePool = rod.NewPagePool(c.parallelism)
}

func (c *Collector) createPage() (*xbot.Bot, *rod.Page) {
	log.Trace().Msg("try get page")

	bot := c.createBot()

	page := c.pagePool.Get(func() *rod.Page {
		page := xbot.CustomizePage(bot.Brw, xbot.Incognito(true))
		return page
	})

	defer c.botPool.Put(bot)
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

	bof := []xbot.BotOptFunc{
		xbot.BotSpawn(false),
		xbot.BotScreen(0),
		xbot.BotHeadless(c.headless),
		xbot.BotUserAgent(c.userAgent),
		xbot.BotProxyServer(proxy),
	}

	return xbot.NewBot(bof...)
}

func (c *Collector) getCookieName(url string) (string, error) {
	name, err := FilenameFromUrl(url)
	if err != nil {
		return "", err
	}

	return path.Join(c.cookieDir, name) + ".cookie.json", nil
}
