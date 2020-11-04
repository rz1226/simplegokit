package kits

import (
	"strconv"
	"sync/atomic"
)

type Counter struct {
	value int64
}

func NewCounter() *Counter {
	return new(Counter)
}

func (c *Counter) Add(i int64) {
	atomic.AddInt64(&c.value, i)
}

func (c *Counter) Count() int64 {
	res := atomic.LoadInt64(&c.value)
	return res
}

func (c *Counter) Get() int64 {
	res := atomic.LoadInt64(&c.value)
	return res
}

func (c *Counter) Str() string {
	return strconv.FormatInt(c.Get(), 10)
}
