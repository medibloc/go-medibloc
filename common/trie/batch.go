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
func (t *Batch) BeginBatch() error {
	if t.batching {
		return ErrBeginAgainInBatch
	}
	t.dirties = make(map[string]*dirty)
	//t.oldRootHash = t.trie.RootHash()
	t.batching = true
	return nil
}

// Clone clone Batch
func (t *Batch) Clone() (*Batch, error) {
	if t.batching {
		return nil, ErrCannotCloneOnBatching
	}
	return NewBatch(t.trie.RootHash(), t.trie.storage)
}

// Commit commit batch WARNING: not thread-safe
func (t *Batch) Commit() error {
	if !t.batching {
		return ErrNotBatching
	}

	for keyHex, d := range t.dirties {
		key := byteutils.Hex2Bytes(keyHex)
		if d.deleteFlag {
			err := t.trie.Delete(key)
			if err != nil && err != ErrNotFound{
				return err
			}
			continue
		}
		if err := t.trie.Put(key, d.value); err != nil {
			return err
		}
	}
	t.dirties = make(map[string]*dirty)
	t.batching = false
	return nil
}

// Delete delete in trie
func (t *Batch) Delete(key []byte) error {
	if !t.batching {
		return ErrNotBatching
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	keyHex := byteutils.Bytes2Hex(key)

	d, ok := t.dirties[keyHex]
	if ok {
		if d.deleteFlag {
			return ErrNotFound
		}
		d.deleteFlag = true
		return nil
	}

	_, err := t.trie.Get(key)
	if err != nil {
		return err
	}
	t.dirties[keyHex] = &dirty{
		value:      nil,
		deleteFlag: true,
	}
	return nil
}

// Get get from trie
func (t *Batch) Get(key []byte) ([]byte, error) {
	if !t.batching {
		return t.trie.Get(key)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	keyHex := byteutils.Bytes2Hex(key)
	d, ok := t.dirties[keyHex]
	if ok {
		if d.deleteFlag {
			return nil, ErrNotFound
		}
		return d.value, nil
	}
	return t.trie.Get(key)
}

// Iterator iterates trie.
func (t *Batch) Iterator(prefix []byte) (*Iterator, error) {
	return t.trie.Iterator(prefix)
}

// Put put to trie
func (t *Batch) Put(key []byte, value []byte) error {
	if !t.batching {
		return ErrNotBatching
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	keyHex := byteutils.Bytes2Hex(key)
	t.dirties[keyHex] = &dirty{
		value:      value,
		deleteFlag: false,
	}

	return nil
}

// RollBack rollback batch WARNING: not thread-safe
func (t *Batch) RollBack() error {
	if !t.batching {
		return ErrNotBatching
	}
	t.dirties = make(map[string]*dirty)
	t.batching = false
	return nil
}

// RootHash getter for rootHash
func (t *Batch) RootHash() ([]byte, error) {
	if t.batching {
		return nil, ErrNotBatching
	}
	return t.trie.RootHash(), nil
}
