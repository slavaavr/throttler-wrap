package throttler_wrap

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ThrottlerTransport struct {
	transport           http.RoundTripper
	reqLimit            uint32
	duration            time.Duration
	exceptUrlPaths      []string
	isRetErrOnOverLimit bool
	whenElapsedCh       <-chan time.Time
	reqCounter          uint32
	barrier             *Barrier
	once                *sync.Once
}

const ReqOverLimitError = "you have exceeded requests count"

func (t *ThrottlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.reqLimit == 0 || t.isExceptUrlPath(req.URL.Path) {
		return t.transport.RoundTrip(req)
	}
	t.once.Do(func() {
		go t.updateThrottlerState()
	})
	if atomic.LoadUint32(&t.reqCounter) == t.reqLimit {
		if t.isRetErrOnOverLimit {
			return nil, errors.New(ReqOverLimitError)
		}
	}
	t.barrier.Acquire()
	atomic.AddUint32(&t.reqCounter, 1)

	return t.transport.RoundTrip(req)
}

func (t *ThrottlerTransport) updateThrottlerState() {
	<-time.After(t.duration)
	t.reqCounter = 0
	acquiredCount := t.barrier.Reset()
	if acquiredCount > 0 {
		t.updateThrottlerState()
	} else {
		t.once = &sync.Once{}
	}
}

func (t ThrottlerTransport) isExceptUrlPath(path string) bool {
	exceptUrlCount := len(t.exceptUrlPaths)
	for k, exceptPath := range t.exceptUrlPaths {
		pathLen := len(path)
		exceptPathLen := len(exceptPath)
		if exceptPath[exceptPathLen-1] == '/' {
			pathLen = strings.LastIndexByte(path, '/')
			exceptPathLen--
		} else if queryIndex := strings.LastIndexByte(path, '?'); queryIndex != -1 {
			pathLen = queryIndex
		}
		if pathLen >= exceptPathLen {
			j := 0
			for i := 0; i < pathLen; i++ {
				if j >= exceptPathLen {
					break
				}
				if path[i] != exceptPath[j] {
					if exceptPath[j] == '*' {
						for i < pathLen && path[i] != '/' {
							i++
						}
						if i == pathLen && j == exceptPathLen-1 { // pattern a/b/.../*
							return true
						}
						j++ // move to /
					} else if k < exceptUrlCount {
						break
					} else {
						return false
					}
				} else if i == pathLen-1 && j == exceptPathLen-1 {
					return true
				}
				j++
			}
		}
	}
	return false
}

func NewThrottler(transport http.RoundTripper, reqLimit uint32, d time.Duration, exceptUrlPaths []string, isRetErrOnOverLimit bool) *ThrottlerTransport {
	var resExceptUrlPaths = exceptUrlPaths
	if resExceptUrlPaths == nil {
		resExceptUrlPaths = []string{}
	}
	return &ThrottlerTransport{
		transport:           transport,
		reqLimit:            reqLimit,
		duration:            d,
		exceptUrlPaths:      resExceptUrlPaths,
		isRetErrOnOverLimit: isRetErrOnOverLimit,

		barrier:    NewBarrier(reqLimit),
		once:       &sync.Once{},
		reqCounter: 0,
	}
}
