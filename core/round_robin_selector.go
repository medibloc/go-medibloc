package core

import (
	"container/list"
	"sync"
)

type rrItem struct {
	key string
	include bool
}

type roundRobin struct {
	sync.Mutex
	incl *list.List
	excl *list.List
	lookup map[string]*list.Element
}

func newRoundRobin() *roundRobin {
	return &roundRobin{
		incl: list.New(),
		excl: list.New(),
		lookup: make(map[string]*list.Element),
	}
}

func (rr *roundRobin) HasNext() bool {
	rr.Lock()
	defer rr.Unlock()
	return rr.incl.Len() != 0
}

func (rr *roundRobin) Next() string {
	rr.Lock()
	defer rr.Unlock()
	e := rr.incl.Front()
	if e == nil {
		return ""
	}
	rr.incl.MoveToBack(e)
	return e.Value.(*rrItem).key
}

func (rr *roundRobin) Include(key string) {
	rr.Lock()
	defer rr.Unlock()

	e, exist := rr.lookup[key]
	if !exist {
		e = rr.incl.PushBack(&rrItem{
			key: key,
			include: true,
		})
		rr.lookup[key] = e
		return
	}

	if e.Value.(*rrItem).include {
		return
	}

	rr.swap(key)
}

func (rr *roundRobin) Exclude(key string) {
	rr.Lock()
	defer rr.Unlock()

	e, exist := rr.lookup[key]
	if !exist {
		e = rr.excl.PushBack(&rrItem{
			key: key,
			include: false,
		})
		rr.lookup[key] = e
		return
	}

	if !e.Value.(*rrItem).include {
		return
	}

	rr.swap(key)
}

func (rr *roundRobin) Remove(key string) {
	rr.Lock()
	defer rr.Unlock()

	e, exist := rr.lookup[key]
	if !exist {
		return
	}

	v := e.Value.(*rrItem)
	if v.include {
		rr.incl.Remove(e)
	} else {
		rr.excl.Remove(e)
	}
	delete(rr.lookup, key)

	return
}

func (rr *roundRobin) Reset() {
	rr.Lock()
	defer rr.Unlock()

	rr.incl.PushBackList(rr.excl)
	rr.excl = list.New()
}

func (rr *roundRobin) swap(key string) {
	e, exist := rr.lookup[key]
	if !exist {
		return
	}

	v := e.Value.(*rrItem)
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

