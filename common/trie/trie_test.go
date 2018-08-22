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

package trie_test

import (
	"encoding/hex"
	"testing"

	"bytes"

	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTrie(t *testing.T) {
	type args struct {
		rootHash []byte
	}
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	tests := []struct {
		args    args
		name    string
		storage storage.Storage
		wantErr bool
	}{
		{
			args{nil},
			"test1",
			s,
			false,
		},
		{
			args{[]byte("root")},
			"test2",
			s,
			true,
		},
	}

	for _, t2 := range tests {
		t.Run(t2.name, func(t3 *testing.T) {
			tr, err := trie.NewTrie(t2.args.rootHash, t2.storage)
			assert.Equal(t3, t2.wantErr, err != nil)
			if err == nil {
				assert.Equal(t3, tr.RootHash(), t2.args.rootHash)
			}
		})
	}
}

func TestTrie_Operations(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	tr, err := trie.NewTrie(nil, s)
	assert.Nil(t, err)
	assert.NotNil(t, tr)

	key1, _ := hex.DecodeString("2ed1b10c")
	val1 := []byte("leafmedi1")
	err = tr.Put(key1, val1)
	assert.Nil(t, err)
	curRootHash := []byte{0xe5, 0xa2, 0x8b, 0x11, 0x2e, 0x6f, 0x65, 0x91, 0xbd, 0xa4, 0x37, 0xd3, 0x86, 0xa9, 0x19, 0xd9, 0xb3, 0x5d, 0x9d, 0x86, 0x3d, 0x6e, 0xd6, 0x5b, 0x19, 0x3, 0x6c, 0x7, 0x6f, 0x25, 0xd4, 0x85}
	assert.Equal(t, curRootHash, tr.RootHash())

	val2, err := tr.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, val2, val1)

	err = tr.Delete(key1)
	assert.Nil(t, err)
	assert.Nil(t, tr.RootHash())

	err = tr.Put(key1, val1)
	assert.Nil(t, err)

	key2, _ := hex.DecodeString("2ed1c000")
	val2 = []byte("leafmedi2")
	err = tr.Put(key2, val2)
	assert.Nil(t, err)

	err = tr.Delete(key2)
	assert.Nil(t, err)

	val2, err = tr.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, val2, val1)
	// TODO make it same topology after put & delete
	// assert.Equal(t, curRootHash, tr.RootHash())

	err = tr.Delete(key1)
	assert.Nil(t, err)
	tr.Get(key2)
	assert.Nil(t, tr.RootHash())
}

type kvs struct {
	key []byte
	val []byte
}

func getKVs() []kvs {
	return []kvs{
		{[]byte("aaabbb"), []byte("val1")},
		{[]byte("aaaccc"), []byte("val2")},
		{[]byte("zzzccc"), []byte("val3")},
	}
}

func TestUpdateExtBranch(t *testing.T) {
	kvs := getKVs()
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	tr, err := trie.NewTrie(nil, s)
	assert.Nil(t, err)
	assert.NotNil(t, tr)

	for _, kv := range kvs {
		err = tr.Put(kv.key, kv.val)
		assert.Nil(t, err)
	}
	for _, kv := range kvs {
		val, err := tr.Get(kv.key)
		assert.Nil(t, err)
		assert.EqualValues(t, kv.val, val)
	}
}

func TestTrie_Clone(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	tr1, err := trie.NewTrie(nil, s)
	assert.Nil(t, err)

	tr2, err := tr1.Clone()
	assert.Nil(t, err)

	kvs := getKVs()
	for _, kv := range kvs {
		err = tr1.Put(kv.key, kv.val)
		assert.Nil(t, err)
	}
	assert.Nil(t, tr2.RootHash())
	for _, kv := range kvs {
		err = tr2.Put(kv.key, kv.val)
		assert.Nil(t, err)
	}
	assert.Equal(t, tr1.RootHash(), tr2.RootHash())
}

type testPair struct {
	route []byte // should be even number of elements
	value string
}

func putPairToTrie(t *testing.T, tr *trie.Trie, pair *testPair) {
	key := trie.RouteToKey(pair.route)
	err := tr.Put(key, []byte(pair.value))
	require.NoError(t, err)
}

func getAndEqual(t *testing.T, tr *trie.Trie, pair *testPair) {
	key := trie.RouteToKey(pair.route)
	v, err := tr.Get(key)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte(pair.value), v))
	t.Log(string(v), ":", tr.ShowPath(trie.RouteToKey(pair.route)))

}

func TestTrieBuild(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	testPairs := []*testPair{
		{[]byte{0, 0, 1, 1, 0, 0}, "first node"},
		{[]byte{0, 0, 1, 1}, "shorter node"},
		{[]byte{0, 0, 1, 1, 0, 0, 1, 1}, "longer node"},
	}

	tr, err := trie.NewTrie(nil, s)
	assert.Nil(t, err)

	t.Log("Put first node")
	putPairToTrie(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[0])

	t.Log("Put shorter node")
	putPairToTrie(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])

	root0 := tr.RootHash()

	t.Log("Put longer node")
	putPairToTrie(t, tr, testPairs[2])
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])

	t.Log("Del longer node")
	err = tr.Delete(trie.RouteToKey(testPairs[2].route))
	require.NoError(t, err)
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])

	_, err = tr.Get(trie.RouteToKey(testPairs[2].route))
	require.Equal(t, trie.ErrNotFound, err)

	assert.Equal(t, root0, tr.RootHash())
}

func TestUpdateExt(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	tr, err := trie.NewTrie(nil, s)
	assert.Nil(t, err)

	testPairs := []*testPair{
		{[]byte{0, 1, 2, 3, 4, 5, 6, 0}, "node0"},
		{[]byte{0, 1, 2, 3, 5, 6, 7, 0}, "node1"},
		{[]byte{0, 1, 2, 3}, "node2"},
		{[]byte{0, 1}, "node3"},
	}

	t.Log("Make ext-branch-Two leafs")
	putPairToTrie(t, tr, testPairs[0])
	putPairToTrie(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])

	t.Log("case matchLen == len(ext's path)")
	putPairToTrie(t, tr, testPairs[2])
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])

	t.Log("case matchLen == len(route)")
	putPairToTrie(t, tr, testPairs[3])
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])
	getAndEqual(t, tr, testPairs[3])

	s, err = storage.NewMemoryStorage()
	assert.Nil(t, err)
	tr, err = trie.NewTrie(nil, s)
	assert.Nil(t, err)

	testPairs = []*testPair{
		{[]byte{0, 1, 2, 3, 4, 5, 6, 0}, "node0"},
		{[]byte{0, 1, 2, 3, 5, 6, 7, 0}, "node1"},
		{[]byte{0, 1, 2, 4}, "node2"},
		{[]byte{0, 2, 0, 0}, "node3"},
	}

	t.Log("Make ext-branch-Two leafs")
	putPairToTrie(t, tr, testPairs[0])
	putPairToTrie(t, tr, testPairs[1])
	putPairToTrie(t, tr, testPairs[2])

	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])

	putPairToTrie(t, tr, testPairs[3])
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])
	getAndEqual(t, tr, testPairs[3])

}

func TestTrie_UpdateLeaf(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	tr, err := trie.NewTrie(nil, s)
	assert.Nil(t, err)

	testPairs := []*testPair{
		{[]byte{0, 1, 2, 3, 4, 5, 6, 0}, "oldLeaf"},
		{[]byte{0, 1, 2, 3, 4, 5, 6, 0}, "newLeaf"},
		{[]byte{0, 1}, "shorterLeaf"},
		{[]byte{0, 1, 2, 3, 4, 5, 6, 0, 1, 2, 3, 4}, "longerLeaf"},
	}

	t.Log("Make a leaf")
	putPairToTrie(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[0])

	putPairToTrie(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[1])

	t.Log("Case: shorterLeaf")
	putPairToTrie(t, tr, testPairs[2])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])

	t.Log("Case: longerLeaf")
	putPairToTrie(t, tr, testPairs[3])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])
	getAndEqual(t, tr, testPairs[3])
}

func TestTrie_Delete(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	testPairs := []*testPair{
		{[]byte{0, 0, 0, 1}, "node0"},
		{[]byte{0, 0, 0, 1, 1, 2}, "node1"},
		{[]byte{0, 0, 0, 1, 1, 3}, "node2"},
		{[]byte{0, 0, 0, 1, 1, 5, 5, 5, 1, 0}, "node3"},
		{[]byte{0, 0, 0, 1, 1, 5, 5, 5, 2, 0}, "node4"},
	}

	tr, err := trie.NewTrie(nil, s)
	require.Nil(t, err)
	putPairToTrie(t, tr, testPairs[0])
	putPairToTrie(t, tr, testPairs[1])

	t.Log("0,1 node")
	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])

	t.Log("Del notfound node - branch")
	err = tr.Delete(trie.RouteToKey([]byte{0, 0, 0, 1, 2, 0}))
	require.Equal(t, trie.ErrNotFound, err)

	t.Log("Del notfound node - ext")
	err = tr.Delete(trie.RouteToKey([]byte{0, 0}))
	require.Equal(t, trie.ErrNotFound, err)

	t.Log("0,1 node - 0 node")
	err = tr.Delete(trie.RouteToKey(testPairs[0].route))
	require.NoError(t, err)
	_, err = tr.Get(trie.RouteToKey(testPairs[0].route))
	require.Equal(t, trie.ErrNotFound, err)
	getAndEqual(t, tr, testPairs[1])

	t.Log("0,1 node - 1 node")
	putPairToTrie(t, tr, testPairs[0])
	err = tr.Delete(trie.RouteToKey(testPairs[1].route))
	require.NoError(t, err)
	getAndEqual(t, tr, testPairs[0])
	_, err = tr.Get(trie.RouteToKey(testPairs[1].route))
	require.Equal(t, trie.ErrNotFound, err)

	t.Log("0,1,2 node")
	putPairToTrie(t, tr, testPairs[1])
	putPairToTrie(t, tr, testPairs[2])

	getAndEqual(t, tr, testPairs[0])
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])

	t.Log("0,1,2 node - 0 node")
	err = tr.Delete(trie.RouteToKey(testPairs[0].route))
	require.NoError(t, err)
	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])
	_, err = tr.Get(trie.RouteToKey(testPairs[0].route))
	require.Equal(t, trie.ErrNotFound, err)

	t.Log("1,2,3,4 node")
	putPairToTrie(t, tr, testPairs[3])
	putPairToTrie(t, tr, testPairs[4])

	getAndEqual(t, tr, testPairs[1])
	getAndEqual(t, tr, testPairs[2])
	getAndEqual(t, tr, testPairs[3])
	getAndEqual(t, tr, testPairs[4])

	t.Log("1,2,3,4 node - 1,2 node")
	err = tr.Delete(trie.RouteToKey(testPairs[1].route))
	require.NoError(t, err)
	err = tr.Delete(trie.RouteToKey(testPairs[2].route))
	require.NoError(t, err)

	getAndEqual(t, tr, testPairs[3])
	getAndEqual(t, tr, testPairs[4])
}

func TestIterator(t *testing.T) {
	testPairs := []*testPair{
		{[]byte{0, 0, 1, 2, 3, 4}, "1st node"},
		{[]byte{0, 0, 1, 2, 3, 5, 6, 7}, "2nd node"},
		{[]byte{0, 0, 1, 2, 3, 6}, "3rd node"},
		{[]byte{0, 0, 1, 2}, "4th node"},
		{[]byte{1, 0, 1, 1, 2, 3, 4, 5}, "5th node"},
	}

	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	tr, err := trie.NewTrie(nil, s)
	assert.Nil(t, err)

	for _, pair := range testPairs {
		putPairToTrie(t, tr, pair)
	}

	iter, err := tr.Iterator(trie.RouteToKey([]byte{0, 0, 1, 2}))
	require.NoError(t, err)

	exist, err := iter.Next()
	require.NoError(t, err)
	for exist {
		t.Log(trie.KeyToRoute(iter.Key()), iter.Value())
		exist, err = iter.Next()
		require.NoError(t, err)
	}
}
