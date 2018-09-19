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
	"sync"

	"github.com/gogo/protobuf/proto"

	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

type dirty struct {
	value      []byte
	deleteFlag bool
}

// Batch batch implementation for trie
type Batch struct {
	mu       sync.RWMutex
	batching bool
	trie     *Trie
	dirties  map[string]*dirty
}

// NewBatch return new Batch instance
func NewBatch(rootHash []byte, stor storage.Storage) (*Batch, error) {
	t, err := NewTrie(rootHash, stor)
	if err != nil {
		return nil, err
	}
	return &Batch{
		trie:    t,
		dirties: make(map[string]*dirty),
	}, nil
}

// BeginBatch begin batch
func (b *Batch) BeginBatch() error {
	if b.batching {
		return ErrBeginAgainInBatch
	}
	b.dirties = make(map[string]*dirty)
	b.batching = true

	b.trie.batching = true
	b.trie.refCounter = make(map[string]*tempNode)
	return nil
}

// Clone clone Batch
func (b *Batch) Clone() (*Batch, error) {
	if b.batching {
		return nil, ErrCannotCloneOnBatching
	}
	return NewBatch(b.trie.RootHash(), b.trie.storage)
}

// Commit commit batch WARNING: not thread-safe
func (b *Batch) Commit() error {
	if !b.batching {
		return ErrNotBatching
	}

	for keyHex, d := range b.dirties {
		key := byteutils.Hex2Bytes(keyHex)
		if d.deleteFlag {
			err := b.trie.Delete(key)
			if err != nil && err != ErrNotFound {
				return err
			}
			continue
		}
		if err := b.trie.Put(key, d.value); err != nil {
			return err
		}
	}
	for _, temp := range b.trie.refCounter {
		if temp.saved || temp.refCount < 1 {
			continue
		}
		pb, err := temp.node.toProto()
		if err != nil {
			return err
		}
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		if err := b.trie.storage.Put(temp.node.Hash, bytes); err != nil {
			return err
		}
	}
	b.trie.refCounter = make(map[string]*tempNode)
	b.trie.batching = false

	b.dirties = make(map[string]*dirty)
	b.batching = false
	return nil
}

// Delete delete in trie
func (b *Batch) Delete(key []byte) error {
	if !b.batching {
		return ErrNotBatching
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	keyHex := byteutils.Bytes2Hex(key)

	d, ok := b.dirties[keyHex]
	if ok {
		if d.deleteFlag {
			return ErrNotFound
		}
		d.deleteFlag = true
		return nil
	}

	_, err := b.trie.Get(key)
	if err != nil {
		return err
	}
	b.dirties[keyHex] = &dirty{
		value:      nil,
		deleteFlag: true,
	}
	return nil
}

// Get get from trie
func (b *Batch) Get(key []byte) ([]byte, error) {
	if !b.batching {
		return b.trie.Get(key)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	keyHex := byteutils.Bytes2Hex(key)
	d, ok := b.dirties[keyHex]
	if ok {
		if d.deleteFlag {
			return nil, ErrNotFound
		}
		return d.value, nil
	}
	return b.trie.Get(key)
}

// Iterator iterates trie.
func (b *Batch) Iterator(prefix []byte) (*Iterator, error) {
	return b.trie.Iterator(prefix)
}

// Put put to trie
func (b *Batch) Put(key []byte, value []byte) error {
	if !b.batching {
		return ErrNotBatching
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	keyHex := byteutils.Bytes2Hex(key)
	b.dirties[keyHex] = &dirty{
		value:      value,
		deleteFlag: false,
	}

	return nil
}

// RollBack rollback batch WARNING: not thread-safe
func (b *Batch) RollBack() error {
	if !b.batching {
		return ErrNotBatching
	}
	b.trie.refCounter = make(map[string]*tempNode)
	b.trie.batching = false

	b.dirties = make(map[string]*dirty)
	b.batching = false
	return nil
}

// RootHash getter for rootHash
func (b *Batch) RootHash() ([]byte, error) {
	if b.batching {
		return nil, ErrNotBatching
	}
	return b.trie.RootHash(), nil
}

func (b *Batch) ShowPath(key []byte) []string {
	return b.trie.ShowPath(key)
}
