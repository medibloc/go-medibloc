package hashheap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type cmp int

func (c cmp) Less(b interface{}) bool {
	return c < b.(cmp)
}

func TestPushPop(t *testing.T) {
	h := NewHashedHeap()
	h.Set("a1", cmp(1))
	h.Set("a2", cmp(2))
	assert.EqualValues(t, 1, h.Pop())
	assert.EqualValues(t, 2, h.Pop())
	assert.Nil(t, h.Pop())

	h.Set("b2", cmp(2))
	h.Set("b1", cmp(1))
	h.Set("b3", cmp(3))
	assert.EqualValues(t, 2, h.Get("b2"))
	assert.EqualValues(t, 3, h.Get("b3"))
	assert.EqualValues(t, 1, h.Get("b1"))
	h.Pop()
	assert.Nil(t, h.Get("b1"))
	assert.EqualValues(t, 2, h.Get("b2"))
	assert.EqualValues(t, 3, h.Get("b3"))
	h.Del("b2")
	h.Del("b2")
	assert.EqualValues(t, 3, h.Pop())
	assert.Nil(t, h.Peek())
	assert.Nil(t, h.Pop())

	h.Set("c1", cmp(1))
	h.Set("c2", cmp(2))
	h.Set("c1", cmp(3))
	assert.EqualValues(t, 2, h.Peek())
	assert.EqualValues(t, 3, h.Get("c1"))
}
