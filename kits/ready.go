package kits

import "sync/atomic"

type Ready struct {
	status uint32 //0表示没有就绪  1就绪
}

func NewReady() *Ready {
	r := &Ready{}
	atomic.StoreUint32(&r.status, 0)
	return r
}
func (r *Ready) SetTrue() {
	atomic.StoreUint32(&r.status, 1)
}
func (r *Ready) SetFalse() {
	atomic.StoreUint32(&r.status, 0)
}

func (r *Ready) IsReady() bool {
	status := atomic.LoadUint32(&r.status)
	if status == 1 {
		return true
	}
	return false
}
