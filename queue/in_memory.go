package queue

import (
	"sync"

	"roddy"
)

// InMemoryQueueStorage is the default implementation of the Storage interface.
// InMemoryQueueStorage holds the request queue in memory.
type InMemoryQueueStorage struct {
	// MaxSize defines the capacity of the queue.
	// New requests are discarded if the queue size reaches MaxSize
	MaxSize int
	lock    *sync.RWMutex
	size    int
	first   *inMemoryQueueItem
	last    *inMemoryQueueItem
}

type inMemoryQueueItem struct {
	Request []byte
	Next    *inMemoryQueueItem
}

func NewInMemory(size int) *InMemoryQueueStorage {
	return &InMemoryQueueStorage{MaxSize: size}
}

func (q *InMemoryQueueStorage) Init() error {
	q.lock = &sync.RWMutex{}

	return nil
}

func (q *InMemoryQueueStorage) AddRequest(r []byte) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.MaxSize > 0 && q.size > q.MaxSize {
		return roddy.ErrQueueFull
	}

	item := &inMemoryQueueItem{Request: r}

	if q.first == nil {
		q.first = item
	} else {
		q.last.Next = item
	}

	q.last = item
	q.size++

	return nil
}

func (q *InMemoryQueueStorage) GetRequest() ([]byte, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.size == 0 {
		return nil, nil
	}

	r := q.first.Request
	q.first = q.first.Next
	q.size--

	return r, nil
}

func (q *InMemoryQueueStorage) QueueSize() (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.size, nil
}
