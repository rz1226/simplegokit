package valuekit

import "sync"

/*
带锁的value
c := NewValueKit()
c.Set(value)
c.Get()
*/

type ValueKit struct {
	mu    *sync.RWMutex
	value interface{}
}

func NewValueKit() *ValueKit {
	v := &ValueKit{}
	v.mu = &sync.RWMutex{}
	return v
}
func (v *ValueKit) Set(value interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.value = value
}
func (v *ValueKit) Get() interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.value
}

type BoolKit struct {
	value *ValueKit
}

func NewBoolKit() *BoolKit {
	b := &BoolKit{}
	b.value = NewValueKit()
	return b
}
func (b *BoolKit) Set(value bool) {
	b.value.Set(value)
}

func (b *BoolKit) Get() bool {
	result := b.value.Get()
	res, ok := result.(bool)
	if !ok {
		return false
	}
	return res
}
