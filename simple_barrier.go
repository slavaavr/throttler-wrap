package throttler_wrap

import (
	"container/list"
	"sync"
)

type Barrier struct {
	size    uint32
	cur     uint32
	mu      sync.Mutex
	waiters list.List
}

func NewBarrier(n uint32) *Barrier {
	return &Barrier{
		size: n,
		cur:  0,
	}
}

func (s *Barrier) Acquire() {
	s.mu.Lock()
	if s.cur < s.size {
		s.cur++
		s.mu.Unlock()
		return
	}
	waiter := make(chan struct{})
	s.waiters.PushBack(waiter)
	s.mu.Unlock()

	select {
	case <-waiter:
	}
}

func (s *Barrier) Reset() uint32 {
	s.mu.Lock()
	s.cur = 0
	var i uint32
	for i = 0; i < s.size; i++ {
		waiter := s.waiters.Front()
		if waiter == nil {
			break
		}
		s.cur++
		wCh := s.waiters.Remove(waiter).(chan struct{})
		close(wCh)
	}
	s.mu.Unlock()
	return s.cur
}
