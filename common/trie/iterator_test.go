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

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common/trie/pb"
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
	for _, v := range names {
		key := byteutils.FromHex(v)
		keys = append(keys, key)
		t.Log(KeyToRoute(key))
	}

	tr.Put(keys[0], []byte(names[0]))
	leaf1 := [][]byte{KeyToRoute(keys[0]), []byte(names[0])}
	leaf1IR, _ := proto.Marshal(&triepb.Node{Type: uint32(leaf), Val: leaf1})
	leaf1H := hash.Sha3256(leaf1IR)
	assert.Equal(t, leaf1H, tr.rootHash)

	tr.Put(keys[1], []byte(names[1]))
	leaf2 := [][]byte{KeyToRoute(keys[0])[4:], []byte(names[0])}
	leaf2IR, _ := proto.Marshal(&triepb.Node{Type: uint32(leaf), Val: leaf2})
	leaf2H := hash.Sha3256(leaf2IR)
	leaf3 := [][]byte{KeyToRoute(keys[1])[4:], []byte(names[1])}
	leaf3IR, _ := proto.Marshal(&triepb.Node{Type: uint32(leaf), Val: leaf3})
	leaf3H := hash.Sha3256(leaf3IR)
	branch2 := [][]byte{nil, nil, nil, leaf3H, leaf2H, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}
	branch2IR, _ := proto.Marshal(&triepb.Node{Type: uint32(branch), Val: branch2})
	branch2H := hash.Sha3256(branch2IR)
	ext1 := [][]byte{KeyToRoute(keys[0])[:3], branch2H}
	ext1IR, _ := proto.Marshal(&triepb.Node{Type: uint32(ext), Val: ext1})
	ext1H := hash.Sha3256(ext1IR)
	assert.Equal(t, ext1H, tr.RootHash())

	tr.Put(keys[2], []byte(names[2]))
	leaf4 := [][]byte{KeyToRoute(keys[2])[3:], []byte(names[2])}
	leaf4IR, _ := proto.Marshal(&triepb.Node{Type: uint32(leaf), Val: leaf4})
	leaf4H := hash.Sha3256(leaf4IR)
	branch4 := [][]byte{nil, nil, leaf4H, branch2H, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}
	branch4IR, _ := proto.Marshal(&triepb.Node{Type: uint32(branch), Val: branch4})
	branch4H := hash.Sha3256(branch4IR)
	ext2 := [][]byte{KeyToRoute(keys[2])[:2], branch4H}
	ext2IR, _ := proto.Marshal(&triepb.Node{Type: uint32(ext), Val: ext2})
	ext2H := hash.Sha3256(ext2IR)
	assert.Equal(t, ext2H, tr.rootHash)

	tr.Put(keys[3], []byte(names[3]))
	leaf5 := [][]byte{KeyToRoute(keys[3])[1:], []byte(names[3])}
	leaf5IR, _ := proto.Marshal(&triepb.Node{Type: uint32(leaf), Val: leaf5})
	leaf5H := hash.Sha3256(leaf5IR)
	ext3 := [][]byte{KeyToRoute(keys[0])[1:2], branch4H}
	ext3IR, _ := proto.Marshal(&triepb.Node{Type: uint32(ext), Val: ext3})
	ext3H := hash.Sha3256(ext3IR)
	branch5 := [][]byte{nil, ext3H, leaf5H, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}
	branch5IR, _ := proto.Marshal(&triepb.Node{Type: uint32(branch), Val: branch5})
	branch5H := hash.Sha3256(branch5IR)
	assert.Equal(t, branch5H, tr.rootHash)

	tr.Put(keys[4], []byte(names[4]))
	leaf6 := [][]byte{KeyToRoute(keys[4])[2:], []byte(names[4])}
	leaf6IR, _ := proto.Marshal(&triepb.Node{Type: uint32(leaf), Val: leaf6})
	leaf6H := hash.Sha3256(leaf6IR)
	branch6 := [][]byte{nil, nil, branch4H, leaf6H, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}
	branch6IR, _ := proto.Marshal(&triepb.Node{Type: uint32(branch), Val: branch6})
	branch6H := hash.Sha3256(branch6IR)
	branch5[1] = branch6H
	branch5IR, _ = proto.Marshal(&triepb.Node{Type: uint32(branch), Val: branch5})
	branch5H = hash.Sha3256(branch5IR)
	assert.Equal(t, branch5H, tr.rootHash)

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
		key := byteutils.FromHex(v)
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
