package roddy

import (
	"sync"

	"github.com/coghost/xbot"
)

type BotPool chan *xbot.Bot

type BotPoolManager struct {
	mut sync.Mutex

	BotPool BotPool
}

func NewBotPoolManager(limit int) *BotPoolManager {
	return &BotPoolManager{
		BotPool: NewBotPool(limit),
	}
}

func (m *BotPoolManager) Get(create func() *xbot.Bot) *xbot.Bot {
	m.mut.Lock()
	bot := m.BotPool.Get(create)
	m.mut.Unlock()

	return bot
}

func (m *BotPoolManager) Put(p *xbot.Bot) {
	m.mut.Lock()
	m.BotPool.Put(p)
	m.mut.Unlock()
}

// NewBotPool instance
func NewBotPool(limit int) BotPool {
	pp := make(chan *xbot.Bot, limit)
	for i := 0; i < limit; i++ {
		pp <- nil
	}

	return pp
}

// Get a browser from the pool. Use the BotPool.Put to make it reusable later.
func (bp BotPool) Get(create func() *xbot.Bot) *xbot.Bot {
	p := <-bp
	if p == nil {
		p = create()
	}

	return p
}

// Put a xbot.Bot back to the pool
func (bp BotPool) Put(p *xbot.Bot) {
	bp <- p
}

// Cleanup helper
func (bp BotPool) Cleanup(iteratee func(*xbot.Bot)) {
	for i := 0; i < cap(bp); i++ {
		p := <-bp
		if p != nil {
			iteratee(p)
		}
	}
}
