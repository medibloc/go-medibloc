// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package hashheap_test

import (
	"testing"

	. "github.com/medibloc/go-medibloc/common/hashheap"
	"github.com/stretchr/testify/assert"
)

type cmp int

func (c cmp) Less(b interface{}) bool {
	return c < b.(cmp)
}

func TestPushPop(t *testing.T) {
	h := New()
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
