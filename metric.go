package roddy

import (
	"github.com/gookit/goutil/dump"
)

func (c *Collector) UpdatePageNum(n uint32) {
	// atomic.StoreUint32(&c.pageNum, n)
	c.pageNum = n
	dump.P("set page num", c.pageNum)
}

func (c *Collector) GetPageNum() uint32 {
	// return atomic.LoadUint32(&c.pageNum)
	return c.pageNum
}

// handleMaxPageNum verify if max page number reached or not.
func (c *Collector) handleMaxPageNum() error {
	if c.maxPageNum == 0 {
		return nil
	}

	pn := c.GetPageNum()
	dump.P("handle max page num", pn, c.maxPageNum)

	if pn < c.maxPageNum {
		return nil
	}

	return ErrMaxPageNumReached
}
