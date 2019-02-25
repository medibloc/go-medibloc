package retry

import (
	"testing"
	"time"
)

// T is an interface wrapper around *testing.T
type T interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

// Retry is a structure that sets the retry count and interval.
type Retry struct {
	t        *testing.T
	count    int
	interval time.Duration
}

// New creates new retry structure.
func New(t *testing.T, count int, interval time.Duration) *Retry {
	return &Retry{
		t:        t,
		count:    count,
		interval: interval,
	}
}

// Try tries the given function.
func (r *Retry) Try(fn func(t T)) {
	wrapT := &wrapTestingT{}
	for i := 0; i < r.count-1; i++ {
		wrapT.failed = false
		fn(wrapT)
		if !wrapT.failed {
			return
		}
		time.Sleep(r.interval)
	}
	fn(r.t)
}

type wrapTestingT struct {
	failed bool
}

func (w *wrapTestingT) Errorf(format string, args ...interface{}) {
	w.failed = true
}

func (w *wrapTestingT) FailNow() {
	w.failed = true
}
