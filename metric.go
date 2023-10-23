package roddy

import "sync/atomic"

func (c *Collector) UpdatePageNum(n uint32) {
	atomic.StoreUint32(&c.pageNum, n)
}

func (c *Collector) GetPageNum() uint32 {
	return atomic.LoadUint32(&c.pageNum)
}

// handleMaxPageNum verify if max page number reached or not.
func (c *Collector) handleMaxPageNum() error {
	if c.maxPageNum == 0 {
		return nil
	}

	if pn := c.GetPageNum(); pn < c.maxPageNum {
		return nil
	}

	return ErrMaxPageNumReached
}
