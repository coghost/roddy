package roddy

import (
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/coghost/xlog"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
)

type RoddySuite struct {
	suite.Suite
	ts *httptest.Server
}

func TestRoddy(t *testing.T) {
	suite.Run(t, new(RoddySuite))
}

func (s *RoddySuite) SetupSuite() {
	xlog.InitLog(xlog.WithNoColor(false), xlog.WithLevel(zerolog.InfoLevel))
	s.ts = newTestServer()
}

func (s *RoddySuite) TearDownSuite() {
	s.ts.Close()
}

var newCollectorTests = map[string]func(*RoddySuite){
	"UserAgent": func(s *RoddySuite) {
		for _, ua := range []string{
			"foo",
			"bar",
		} {
			c := NewCollector(UserAgent(ua))
			s.Equal(ua, c.userAgent, "want "+ua)
		}
	},
	"MaxDepth": func(s *RoddySuite) {
		for _, depth := range []int{
			12,
			34,
			0,
		} {
			c := NewCollector(MaxDepth(depth))
			s.Equal(depth, c.maxDepth)
		}
	},
	"Headless": func(s *RoddySuite) {
		for _, b := range []bool{
			false,
			true,
		} {
			c := NewCollector(Headless(b))
			s.Equal(b, c.headless)
		}
	},
	"AllowedDomains": func(s *RoddySuite) {
		for _, domains := range [][]string{
			{"example.com", "example.net"},
			{"example.net"},
			{},
			nil,
		} {
			c := NewCollector(AllowedDomains(domains...))
			s.Equal(domains, c.allowedDomains)
		}
	},
	"DisallowedDomains": func(s *RoddySuite) {
		for _, domains := range [][]string{
			{"example.com", "example.net"},
			{"example.net"},
			{},
			nil,
		} {
			c := NewCollector(DisallowedDomains(domains...))
			s.Equal(domains, c.disallowedDomains)
		}
	},
	"DisallowedURLFilters": func(s *RoddySuite) {
		for _, filters := range [][]*regexp.Regexp{
			{regexp.MustCompile(`.*not_allowed.*`)},
		} {
			c := NewCollector(DisallowedURLFilters(filters...))

			s.Equal(filters, c.disallowedURLFilters)
		}
	},
	"URLFilters": func(s *RoddySuite) {
		for _, filters := range [][]*regexp.Regexp{
			{regexp.MustCompile(`\w+`)},
			{regexp.MustCompile(`\d+`)},
			{},
			nil,
		} {
			c := NewCollector(URLFilters(filters...))
			s.Equal(filters, c.urlFilters)
		}
	},
}

func (s *RoddySuite) Test_00_NewCollector() {
	for _, tt := range newCollectorTests {
		tt(s)
	}
}

func (s *RoddySuite) Test_11_Visit() {
	c := NewCollector()

	onRequestCalled := false
	onResponseCalled := false

	c.OnRequest(func(r *Request) {
		onRequestCalled = true
		r.Ctx.Put("x", "y")
	})

	c.OnResponse(func(r *Response) {
		onResponseCalled = true
		s.Equal("y", r.Request.Ctx.Get("x"))
	})

	c.Visit(s.ts.URL)

	s.True(onRequestCalled, "request should be called")
	s.True(onResponseCalled, "response should be called")
}

func (s *RoddySuite) Test_12_VisitAllowedDomains() {
	// TODO:
}

func (s *RoddySuite) Test_13_VisitDisallowedDomains() {
	// TODO:
}

func (s *RoddySuite) Test_20_OnHTML() {
	c := NewCollector()

	titleCallbackCalled := false
	pTagCallbackCount := 0

	c.OnHTML("title", func(e *HTMLElement) {
		titleCallbackCalled = true
		s.Equal("Test Page", e.Text(), "Title element text")
	})

	c.OnHTML("p", func(e *HTMLElement) {
		pTagCallbackCount++
		s.Equal("description", e.Attr("class"))
	})

	c.OnHTML("body", func(e *HTMLElement) {
		s.Equal("description", *e.DOM.MustElement("p").MustAttribute("class"))
		s.Equal(2, len(e.DOM.MustElements("p")))
	})

	c.Visit(s.ts.URL + "/html")

	s.True(titleCallbackCalled, "call OnHTML callback")
	s.Equal(2, pTagCallbackCount, "find all <p> tags")
}

func (s *RoddySuite) Test_30_Depth() {
	maxDepth := 2

	c1 := NewCollector(MaxDepth(maxDepth), AllowURLRevisit(true))
	requestCount := 0
	c1.OnResponse(func(r *Response) {
		requestCount++
		if requestCount >= 10 {
			return
		}
		err := c1.Visit(s.ts.URL)
		s.Nil(err)
	})
	c1.Visit(s.ts.URL)
	s.LessOrEqual(10, requestCount, "max depth is not worked.")
}
