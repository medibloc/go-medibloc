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

package trie

import (
	"testing"

	"github.com/stretchr/testify/require"

	"fmt"

	"github.com/gogo/protobuf/proto"
	triepb "github.com/medibloc/go-medibloc/common/trie/pb"
	"github.com/medibloc/go-medibloc/crypto/hash"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/stretchr/testify/assert"
)

func TestIterator1(t *testing.T) {
	stor, _ := storage.NewMemoryStorage()
	tr, err := NewTrie(nil, stor)
	assert.Nil(t, err)
	names := []string{"123450", "123350", "122450", "223350", "133350"}
	var keys [][]byte
	var valHashes [][]byte
	for _, v := range names {
		key, err := byteutils.FromHex(v)
		require.NoError(t, err)
		keys = append(keys, key)

		value := [][]byte{[]byte(v)}
		valueByte, _ := proto.Marshal(&triepb.Node{Type: uint32(val), Val: value})
		valHash := hash.Sha3256(valueByte)
		valHashes = append(valHashes, valHash)
	}

	tr.Put(keys[0], []byte(names[0]))
	t.Log("Put 1st node")
	t.Log(KeyToRoute(keys[0]), ":", tr.ShowPath(keys[0]))
	assert.Equal(t, "[ext([1 2 3 4 5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[0])))

	t.Log("Put 2nd node")
	tr.Put(keys[1], []byte(names[1]))
	t.Log(KeyToRoute(keys[0]), ":", tr.ShowPath(keys[0]))
	t.Log(KeyToRoute(keys[1]), ":", tr.ShowPath(keys[1]))
	assert.Equal(t, "[ext([1 2 3]) branch(4) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[0])))
	assert.Equal(t, "[ext([1 2 3]) branch(3) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[1])))

	t.Log("Put 3rd node")
	tr.Put(keys[2], []byte(names[2]))
	t.Log(KeyToRoute(keys[0]), ":", tr.ShowPath(keys[0]))
	t.Log(KeyToRoute(keys[1]), ":", tr.ShowPath(keys[1]))
	t.Log(KeyToRoute(keys[2]), ":", tr.ShowPath(keys[2]))

	assert.Equal(t, "[ext([1 2]) branch(3) branch(4) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[0])))
	assert.Equal(t, "[ext([1 2]) branch(3) branch(3) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[1])))
	assert.Equal(t, "[ext([1 2]) branch(2) ext([4 5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[2])))

	t.Log("Put 4th node")
	tr.Put(keys[3], []byte(names[3]))
	t.Log(KeyToRoute(keys[0]), ":", tr.ShowPath(keys[0]))
	t.Log(KeyToRoute(keys[1]), ":", tr.ShowPath(keys[1]))
	t.Log(KeyToRoute(keys[2]), ":", tr.ShowPath(keys[2]))
	t.Log(KeyToRoute(keys[3]), ":", tr.ShowPath(keys[3]))

	assert.Equal(t, "[branch(1) ext([2]) branch(3) branch(4) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[0])))
	assert.Equal(t, "[branch(1) ext([2]) branch(3) branch(3) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[1])))
	assert.Equal(t, "[branch(1) ext([2]) branch(2) ext([4 5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[2])))
	assert.Equal(t, "[branch(2) ext([2 3 3 5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[3])))

	t.Log("Put 5th node")
	tr.Put(keys[4], []byte(names[4]))
	t.Log(KeyToRoute(keys[0]), ":", tr.ShowPath(keys[0]))
	t.Log(KeyToRoute(keys[1]), ":", tr.ShowPath(keys[1]))
	t.Log(KeyToRoute(keys[2]), ":", tr.ShowPath(keys[2]))
	t.Log(KeyToRoute(keys[3]), ":", tr.ShowPath(keys[3]))
	t.Log(KeyToRoute(keys[4]), ":", tr.ShowPath(keys[4]))

	assert.Equal(t, "[branch(1) branch(2) branch(3) branch(4) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[0])))
	assert.Equal(t, "[branch(1) branch(2) branch(3) branch(3) ext([5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[1])))
	assert.Equal(t, "[branch(1) branch(2) branch(2) ext([4 5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[2])))
	assert.Equal(t, "[branch(2) ext([2 3 3 5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[3])))
	assert.Equal(t, "[branch(1) branch(3) ext([3 3 5 0]) val]", fmt.Sprintf("%v", tr.ShowPath(keys[4])))

	it, err := tr.Iterator([]byte{0x12})
	assert.Nil(t, err)
	next, err := it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[2]))
	assert.Equal(t, it.Key(), []byte(keys[2]))

	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[1]))
	assert.Equal(t, it.Key(), []byte(keys[1]))

	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[0]))
	assert.Equal(t, it.Key(), []byte(keys[0]))

	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, false)
}

func TestIterator2(t *testing.T) {
	stor, _ := storage.NewMemoryStorage()
	tr, err := NewTrie(nil, stor)
	assert.Nil(t, err)
	names := []string{"123450", "123350", "122450", "223350", "133350"}
	var keys [][]byte
	for _, v := range names {
		key, err := byteutils.FromHex(v)
		require.NoError(t, err)
		keys = append(keys, key)
	}
	tr.Put(keys[0], []byte(names[0]))

	iter, err1 := tr.Iterator([]byte{0x12, 0x34, 0x50, 0x12})
	assert.Nil(t, err1)
	exist, err := iter.Next()
	assert.Nil(t, err)
	assert.False(t, exist)

	it, err := tr.Iterator([]byte{0x12})
	assert.Nil(t, err)

	next, err := it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[0]))
	assert.Equal(t, it.Key(), []byte(keys[0]))

	tr.Put(keys[1], []byte(names[1]))
	it, err = tr.Iterator([]byte{0x12})
	assert.Nil(t, err)
	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[1]))
	assert.Equal(t, it.Key(), []byte(keys[1]))
	next, err = it.Next()
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[0]))
	assert.Equal(t, it.Key(), []byte(keys[0]))

	tr.Put(keys[2], []byte(names[2]))

	it, err = tr.Iterator(nil)
	assert.Nil(t, err)
	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[2]))
	assert.Equal(t, it.Key(), []byte(keys[2]))

	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[1]))
	assert.Equal(t, it.Key(), []byte(keys[1]))

	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, true)
	assert.Equal(t, it.Value(), []byte(names[0]))
	assert.Equal(t, it.Key(), []byte(keys[0]))

	next, err = it.Next()
	assert.Nil(t, err)
	assert.Equal(t, next, false)
}
