package roddy

import (
	"path"

	"github.com/coghost/xbot"
	"github.com/coghost/xutil"
	"github.com/go-rod/rod"
	"github.com/gookit/goutil/arrutil"
	"github.com/rs/zerolog/log"
	"github.com/ungerik/go-dry"
)

func (c *Collector) MustGoBack(args ...*rod.Page) {
	if c.maxDepth == 0 {
		return
	}

	page := c.Bot.Pg
	if len(args) > 0 {
		page = args[1]
	}

	err := page.NavigateBack()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot go back")
	}

	page.MustWaitLoad()
}

func (c *Collector) DumpCookies() error {
	ck, err := c.getCookieName(c.Bot.CurrentUrl())
	if err != nil {
		return err
	}

	return dry.FileSetJSON(ck, c.Bot.Pg.MustCookies())
}

func (c *Collector) SetCookies(url string, page *rod.Page) error {
	ck, err := c.getCookieName(url)
	if err != nil {
		return err
	}

	raw, err := dry.FileGetString(ck)
	if err != nil {
		return err
	}

	return c.Bot.SetPageWithCookies(page, raw)
}

func (c *Collector) initDefaultBot() {
	c.initPagePool()

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

	c.Bot = xbot.NewBot(bof...)
}

func (c *Collector) initPagePool() {
	if !c.async {
		return
	}

	// if async mode, force set Parallelism to 1, if is 0
	c.parallelism = xutil.AorB(c.parallelism, 1)
	c.pagePool = rod.NewPagePool(c.parallelism)
}

func (c *Collector) createBot() {
	if c.Bot.Brw != nil {
		return
	}

	log.Trace().Msg("no bot found, create bot")
	xbot.Spawn(c.Bot)

	// in async mode, after spawn browser and page, put page to pool
	if c.async {
		log.Trace().Msg("put default page to page pool")

		pg := c.pagePool.Get(func() *rod.Page {
			return c.Bot.Pg
		})
		c.pagePool.Put(pg)
	}
}

func (c *Collector) createPage() *rod.Page {
	if !c.async {
		return c.Bot.Pg
	}

	brw := c.Bot.Brw

	return c.pagePool.Get(func() *rod.Page {
		// to run page paralell, incognito mode required.
		page := xbot.CustomizePage(brw, xbot.Incognito(true))
		return page
	})
}

func (c *Collector) getCookieName(url string) (string, error) {
	name, err := FilenameFromUrl(url)
	if err != nil {
		return "", err
	}

	return path.Join(c.cookieDir, name) + ".cookie.json", nil
}
