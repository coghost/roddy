package main

import (
	"fmt"
	"os"
	"time"

	"roddy/queue"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/k0kubun/pp/v3"
	"github.com/rs/zerolog/log"
)

type item struct {
	StoryURL  string
	Source    string
	CrawledAt time.Time
	Comments  string
	Title     string
}

func runWithQueue() {
	var stories []item

	xlog.InitLogDebug()

	cap_ := 2
	q, _ := queue.New(cap_, queue.NewInMemory(10000))

	c := roddy.NewCollector(
		roddy.AllowedDomains("old.reddit.com"),
		roddy.Parallelism(cap_),
		roddy.RandomDelay(50*time.Millisecond),
		// roddy.MaxDepth(5),
		roddy.MaxResponse(5),
	)

	c.OnData(`div.top-matter`, func(e *roddy.DataElement) {
		story := item{}
		story.StoryURL = e.ChildAttr("a[data-event-action=title]", "href")
		story.Source = "https://old.reddit.com/r/gaming/"
		story.Title = e.ChildText("a[data-event-action=title]")
		story.Comments = e.ChildAttr("a[data-event-action=comments]", "href")
		story.CrawledAt = time.Now()

		stories = append(stories, story)
	})

	c.OnPaging(`span.next-button>a`, func(e *roddy.SerpElement) {
		fmt.Println("got", len(stories), e.Link())
		// e.Click(e.Selector)
		e.Request.Visit(e.Link())
	})

	const base = "https://old.reddit.com/r/%s/"

	for _, category := range os.Args[1:] {
		q.AddURL(fmt.Sprintf(base, category))
	}

	q.Run(c)

	fmt.Println(c.String(), len(stories))
}

func runAsync() {
	var stories []item

	xlog.InitLogDebug()

	cap_ := 2

	c := roddy.NewCollector(
		roddy.AllowedDomains("old.reddit.com"),
		roddy.MaxDepth(5),
		roddy.Async(true),
		roddy.Parallelism(cap_),
		roddy.RandomDelay(10*time.Millisecond),
	)

	c.OnData(`div.top-matter`, func(e *roddy.DataElement) {
		story := item{}
		story.StoryURL = e.ChildAttr("a[data-event-action=title]", "href")
		story.Source = e.Request.URL.String()
		story.Title = e.ChildText("a[data-event-action=title]")
		story.Comments = e.ChildAttr("a[data-event-action=comments]", "href")
		story.CrawledAt = time.Now()
		stories = append(stories, story)
	})

	c.OnPaging(`span.next-button`, func(e *roddy.SerpElement) {
		fmt.Println("got", len(stories), e.Request.String())
		e.Click(e.Selector)
		e.Request.VisitByMockClick()
	})

	c.OnError(func(r *roddy.Response, err error) {
		pp.Println(err)
	})

	const base = "https://old.reddit.com/r/%s/"

	for _, category := range os.Args[1:] {
		c.Visit(fmt.Sprintf(base, category))
	}

	c.Wait()

	log.Info().Int("stories", len(stories)).Msg(c.String())
}

func main() {
	runWithQueue()
	// runAsync()
}
