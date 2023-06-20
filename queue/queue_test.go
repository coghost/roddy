package queue

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"roddy"

	"github.com/coghost/xlog"
	"github.com/gookit/goutil/dump"
	"github.com/stretchr/testify/suite"
)

type QueueSuite struct {
	suite.Suite
}

func TestQueue(t *testing.T) {
	suite.Run(t, new(QueueSuite))
}

func (s *QueueSuite) SetupSuite() {
	xlog.InitLogForConsole()
}

func (s *QueueSuite) TearDownSuite() {
}

func (s *QueueSuite) Test_Queue() {
	server := httptest.NewServer(http.HandlerFunc(serverHandler))
	defer server.Close()

	rng := rand.New(rand.NewSource(12387123712321232))
	var (
		items    uint32
		requests uint32
		success  uint32
		failure  uint32
	)

	storage := NewInMemory(100)

	q, err := New(2, storage)
	if err != nil {
		panic(err)
	}

	put := func() {
		t := time.Duration(rng.Intn(50)) * time.Microsecond
		url := server.URL + "/delay?t=" + t.String()
		atomic.AddUint32(&items, 1)
		q.AddURL(url)
	}

	for i := 0; i < 30; i++ {
		put()
		storage.AddRequest([]byte("error request"))
	}

	c := roddy.NewCollector(
		roddy.AllowURLRevisit(true),
		roddy.RandomDelay(100*time.Millisecond),
	)

	c.OnRequest(func(r *roddy.Request) {
		atomic.AddUint32(&requests, 1)
	})

	c.OnResponse(func(r *roddy.Response) {
		if r.Page.String() != "" {
			atomic.AddUint32(&success, 1)
		} else {
			atomic.AddUint32(&failure, 1)
		}

		toss := rng.Intn(2) == 0
		if toss {
			put()
		}
	})

	c.OnError(func(r *roddy.Response, err error) {
		atomic.AddUint32(&failure, 1)
	})

	err = q.Run(c)
	s.Nil(err, "Queue.Run() returns no error")

	s.Equal(items, requests, "items equal with requests")
	s.Equal(success+failure, requests, "success+failure equal with requests")
	s.Greater(failure, uint32(0), "has failures")

	dump.P(items, requests, success, failure)
}

func serverHandler(w http.ResponseWriter, req *http.Request) {
	if !serverRoute(w, req) {
		shutdown(w)
	}
}

func serverRoute(w http.ResponseWriter, req *http.Request) bool {
	if req.URL.Path == "/delay" {
		return serverDelay(w, req) == nil
	}

	return false
}

func serverDelay(w http.ResponseWriter, req *http.Request) error {
	q := req.URL.Query()
	t, err := time.ParseDuration(q.Get("t"))
	if err != nil {
		return err
	}

	time.Sleep(t)
	w.WriteHeader(http.StatusOK)

	return nil
}

func shutdown(w http.ResponseWriter) {
	taker, ok := w.(http.Hijacker)
	if !ok {
		return
	}

	raw, _, err := taker.Hijack()
	if err != nil {
		return
	}

	raw.Close()
}
