package ratekit

/*
计算最近收集的数字的和
*/

import (
	//"fmt"
	"sync"
)

type RecentSum struct {
	ints []int
	len  int
	mu   *sync.RWMutex
}

func NewRecentSum(len int) *RecentSum {
	if len <= 0 {
		len = 5
	}
	r := &RecentSum{}
	r.len = len
	r.mu = &sync.RWMutex{}
	r.ints = make([]int, len, len)
	for i := 0; i < len; i++ {
		r.ints[i] = 0
	}
	return r
}

func (r *RecentSum) Put(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ints = append(r.ints[1:r.len], n)

}
func (r *RecentSum) Sum() int {
	//fmt.Println( r.ints )
	r.mu.RLock()
	defer r.mu.RUnlock()
	sum := 0
	for _, e := range r.ints {
		sum += e
	}
	return sum
}
