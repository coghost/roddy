package queue

import (
	"sync"

	"roddy"

	whatwgUrl "github.com/nlnwa/whatwg-url/url"
)

const _stop = true

var urlParser = whatwgUrl.NewParser(whatwgUrl.WithPercentEncodeSinglePercentSign())

// Storage is the interface of the queue's storage backend
// Storage must be concurrently safe for multiple goroutines.
type Storage interface {
	// Init initializes the storage
	Init() error
	// AddRequest adds a serialized request to the queue
	AddRequest([]byte) error
	// GetRequest pops the next request from the queue
	// or returns error if the queue is empty
	GetRequest() ([]byte, error)
	// QueueSize returns with the size of the queue
	QueueSize() (int, error)
}

// Queue is a request queue which uses a Collector to consume
// requests in multiple threads
type Queue struct {
	// Threads defines the number of consumer threads
	Threads int
	storage Storage
	wake    chan struct{}
	mut     sync.Mutex // guards wake and running
	running bool
}

func New(threads int, s Storage) (*Queue, error) {
	if s == nil {
		s = NewInMemory(100000)
	}

	if err := s.Init(); err != nil {
		return nil, err
	}

	return &Queue{
		Threads: threads,
		storage: s,
		running: true,
	}, nil
}

// IsEmpty returns true if the queue is empty.
func (q *Queue) IsEmpty() bool {
	s, _ := q.Size()
	return s == 0
}

// Size returns the size of the queue
func (q *Queue) Size() (int, error) {
	return q.storage.QueueSize()
}

func (q *Queue) AddURL(URL string) error {
	u2, err := roddy.ParseUrl(URL)
	if err != nil {
		return err
	}

	r := &roddy.Request{
		URL: u2,
	}

	return q.storeRequest(r)
}

func (q *Queue) AddRequest(r *roddy.Request) error {
	q.mut.Lock()
	waken := q.wake != nil
	q.mut.Unlock()

	if !waken {
		return q.storeRequest(r)
	}

	err := q.storeRequest(r)
	if err != nil {
		return err
	}

	q.wake <- struct{}{}

	return nil
}

func (q *Queue) storeRequest(r *roddy.Request) error {
	buf, err := r.Marshal()
	if err != nil {
		return err
	}

	return q.storage.AddRequest(buf)
}

func (q *Queue) Run(c *roddy.Collector) error {
	q.mut.Lock()
	if q.wake != nil && q.running == true {
		q.mut.Unlock()
		panic("cannot call duplicate Queue.Run")
	}

	q.wake = make(chan struct{})
	q.running = true
	q.mut.Unlock()

	reqChan := make(chan *roddy.Request)
	complete, errChan := make(chan struct{}), make(chan error, 1)

	for i := 0; i < q.Threads; i++ {
		go independentRunner(reqChan, complete)
	}

	go q.loop(c, reqChan, complete, errChan)

	defer close(reqChan)

	return <-errChan
}

// Stop will stop the running queue
func (q *Queue) Stop() {
	q.mut.Lock()
	q.running = false
	q.mut.Unlock()
}

func (q *Queue) loop(c *roddy.Collector, reqChan chan<- *roddy.Request, complete <-chan struct{}, errc chan<- error) {
	var active int

	for {
		size, err := q.storage.QueueSize()
		if err != nil {
			errc <- err
			break
		}

		if size == 0 && active == 0 || !q.running {
			// Terminate when
			//   1. No active requests
			//   2. Empty queue
			errc <- nil
			break
		}

		req := &roddy.Request{}
		sent := reqChan

		if size > 0 {
			req, err = q.loadRequest(c)
			if err != nil {
				// ignore error returned by GetRequest() or UnmarshalRequest()
				continue
			}
		} else {
			sent = nil
		}

	SENT:
		for {
			select {
			case sent <- req:
				active++
				break SENT
			case <-q.wake:
				if sent == nil {
					break SENT
				}
			case <-complete:
				active--
				if sent == nil && active == 0 {
					break SENT
				}
			}
		}
	}
}

func (q *Queue) loadRequest(c *roddy.Collector) (*roddy.Request, error) {
	buf, err := q.storage.GetRequest()
	if err != nil {
		return nil, err
	}

	copied := make([]byte, len(buf))
	copy(copied, buf)

	return c.UnmarshalRequest(copied)
}

func independentRunner(reqChan <-chan *roddy.Request, complete chan<- struct{}) {
	for req := range reqChan {
		req.Do()
		complete <- struct{}{}
	}
}
