package roundrobin

import (
	"container/list"
	"sync"
)

// RoundRobin manages item in round-robin order.
type RoundRobin struct {
	sync.Mutex
	incl   *list.List
	excl   *list.List
	lookup map[string]*list.Element
}

// New creates RoundRobin.
func New() *RoundRobin {
	return &RoundRobin{
		incl:   list.New(),
		excl:   list.New(),
		lookup: make(map[string]*list.Element),
	}
}

// HasNext returns true if next item exists.
func (rr *RoundRobin) HasNext() bool {
	rr.Lock()
	defer rr.Unlock()
	return rr.incl.Len() != 0
}

// Next returns next item in round-robin order.
func (rr *RoundRobin) Next() string {
	rr.Lock()
	defer rr.Unlock()
	e := rr.incl.Front()
	if e == nil {
		return ""
	}
	rr.incl.MoveToBack(e)
	return e.Value.(*item).key
}

// Include adds a item in include list or swaps a item from  exclude list.
// Only items in include list can't be fetched in round-robin order.
func (rr *RoundRobin) Include(key string) {
	rr.Lock()
	defer rr.Unlock()

	e, exist := rr.lookup[key]
	if !exist {
		e = rr.incl.PushBack(&item{
			key:     key,
			include: true,
		})
		rr.lookup[key] = e
		return
	}

	if e.Value.(*item).include {
		return
	}

	rr.swap(key)
}

// Exclude adds a item in exclude list or swaps a item from include list.
// Items in exclude list are excluded and can't be fetched.
func (rr *RoundRobin) Exclude(key string) {
	rr.Lock()
	defer rr.Unlock()

	e, exist := rr.lookup[key]
	if !exist {
		e = rr.excl.PushBack(&item{
			key:     key,
			include: false,
		})
		rr.lookup[key] = e
		return
	}

	if !e.Value.(*item).include {
		return
	}

	rr.swap(key)
}

// Remove removes a item.
func (rr *RoundRobin) Remove(key string) {
	rr.Lock()
	defer rr.Unlock()

	e, exist := rr.lookup[key]
	if !exist {
		return
	}

	v := e.Value.(*item)
	if v.include {
		rr.incl.Remove(e)
	} else {
		rr.excl.Remove(e)
	}
	delete(rr.lookup, key)

	return
}

// Reset appends items in exclude list to include list and resets exclude list.
func (rr *RoundRobin) Reset() {
	rr.Lock()
	defer rr.Unlock()

	rr.incl.PushBackList(rr.excl)
	rr.excl = list.New()
}

func (rr *RoundRobin) swap(key string) {
	e, exist := rr.lookup[key]
	if !exist {
		return
	}

	v := e.Value.(*item)
	from := rr.incl
	to := rr.excl
	if !v.include {
		from = rr.excl
		to = rr.incl
	}

	from.Remove(e)

	v.include = !v.include
	e2 := to.PushBack(v)
	rr.lookup[key] = e2
}

type item struct {
	key     string
	include bool
}
