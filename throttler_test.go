package throttler_wrap

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestThrottlerQuery(t *testing.T) {
	throttled := NewThrottler(
		http.DefaultTransport,
		60,
		time.Minute,
		[]string{"/servers/*/status", "/network/", "/doodles/"},
		false,
	)
	client := http.Client{
		Transport: throttled,
	}
	resp, err := client.Get("https://www.google.com/doodles/john-lennons-70th-birthday")
	if err != nil {
		t.Errorf("error, %s", err.Error())
	} else if resp.StatusCode != http.StatusOK {
		t.Errorf("error, wrong status code, got %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

func TestThrottlerError(t *testing.T) {
	throttled := NewThrottler(
		http.DefaultTransport,
		5,
		3*time.Second,
		[]string{"/servers/*/status", "/network/"},
		true,
	)
	client := http.Client{
		Transport: throttled,
	}
	var countOfErrors int32
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.Get("https://www.google.com/doodles/john-lennons-70th-birthday")
			if err != nil {
				atomic.AddInt32(&countOfErrors, 1)
			}
		}()
	}
	wg.Wait()
	if countOfErrors != 5 {
		t.Errorf("error, wrong number of err, got %d, expected %d", countOfErrors, 5)
	}
}

func TestThrottler(t *testing.T) {
	throttled := NewThrottler(
		http.DefaultTransport,
		4,
		3*time.Second,
		[]string{"/servers/*/status", "/network/"},
		false,
	)
	client := http.Client{
		Transport: throttled,
	}
	var countOfCompletedFunctions int32
	for i := 0; i < 10; i++ {
		go func() {
			_, err := client.Get("https://www.google.com/doodles/john-lennons-70th-birthday")
			if err == nil {
				atomic.AddInt32(&countOfCompletedFunctions, 1)
			}
		}()
	}
	<-time.After(7 * time.Second)
	if countOfCompletedFunctions != 10 {
		t.Errorf("error, whong count of completed funcs, got %d, expected %d", countOfCompletedFunctions, 10)
	}
}

func TestExceptPath(t *testing.T) {
	throttled := NewThrottler(
		http.DefaultTransport,
		60,
		time.Minute,
		[]string{"/servers/*/status", "/network/", "/*/a/b", "/a/b/*", "/a/*/*/b/*/*/*/c"},
		false,
	)
	tests := []struct {
		actual   string
		expected bool
	}{
		{"/network/routes", true},
		{"/network/routes/123", false},
		{"/images/reload", false},
		{"/servers/1337/status?simple=true&hard=false", true},
		{"/servers/1337/status", true},
		{"/servers/1337/status/test", false},
		{"/test/a/b", true},
		{"/a/b/test", true},
		{"/a/b/c/test", false},
		{"/a/test1/test2/b/test3/test4/test5/c?simple=true", true},
		{"/a/test1/test2/b/test3/test4/test5/c/d?simple=true", false},
		{"/a/test1/b/test3/test4/test5/c/d?simple=true", false},
		{"/", false},
	}
	for _, test := range tests {
		if throttled.isExceptUrlPath(test.actual) != test.expected {
			t.Errorf("error, got %s, expected %t", test.actual, test.expected)
		}
	}

}
