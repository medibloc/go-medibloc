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
	*Trie

	mu       sync.RWMutex
	batching bool
	dirties  map[string]*dirty
}

// NewBatch return new Batch instance
func NewBatch(rootHash []byte, stor storage.Storage) (*Batch, error) {
	t, err := NewTrie(rootHash, stor)
	if err != nil {
		return nil, err
	}
	return &Batch{
		Trie:     t,
		dirties:  make(map[string]*dirty),
		batching: false,
	}, nil
}

// Clone clone Batch
func (tb *Batch) Clone() (*Batch, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.batching {
		return nil, ErrCannotPerformInBatch
	}
	tr, err := tb.Trie.Clone()
	if err != nil {
		return nil, err
	}
	newBatch := &Batch{
		Trie:    tr,
		dirties: make(map[string]*dirty),
	}
	return newBatch, nil
}

// BeginBatch begin batch
func (tb *Batch) BeginBatch() error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.batching {
		return ErrCannotPerformInBatch
	}
	tb.dirties = make(map[string]*dirty)
	tb.batching = true
	return nil
}

// Commit commit batch WARNING: not thread-safe
func (tb *Batch) Commit() error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if !tb.batching {
		return ErrNotBatching
	}
	for keyHex, d := range tb.dirties {
		key, err := byteutils.Hex2Bytes(keyHex)
		if err != nil {
			return err
		}
		if d.deleteFlag {
			err := tb.Trie.Delete(key)
			if err != nil && err != ErrNotFound {
				return err
			}
			continue
		}
		if err := tb.Trie.Put(key, d.value); err != nil {
			return err
		}
	}
	tb.dirties = make(map[string]*dirty)
	tb.batching = false
	return nil
}

// RollBack rollback batch WARNING: not thread-safe
func (tb *Batch) RollBack() error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if !tb.batching {
		return ErrNotBatching
	}

	tb.dirties = make(map[string]*dirty)
	tb.batching = false
	return nil
}

// Delete delete in trie
func (tb *Batch) Delete(key []byte) error {
	tb.mu.Lock()
	if tb.batching {
		err := tb.batchDelete(key)
		tb.mu.Unlock()
		return err
	}
	tb.mu.Unlock()

	return tb.Trie.Delete(key)
}

func (tb *Batch) batchDelete(key []byte) error {
	keyHex := byteutils.Bytes2Hex(key)

	d, ok := tb.dirties[keyHex]
	if ok {
		if d.deleteFlag {
			return ErrNotFound
		}
		d.deleteFlag = true
		return nil
	}

	_, err := tb.Trie.Get(key)
	if err != nil {
		return err
	}
	tb.dirties[keyHex] = &dirty{
		value:      nil,
		deleteFlag: true,
	}
	return nil
}

// Get get from trie
func (tb *Batch) Get(key []byte) ([]byte, error) {
	tb.mu.Lock()
	if tb.batching {
		val, err := tb.batchGet(key)
		tb.mu.Unlock()
		return val, err
	}
	tb.mu.Unlock()

	return tb.Trie.Get(key)
}

func (tb *Batch) batchGet(key []byte) ([]byte, error) {
	keyHex := byteutils.Bytes2Hex(key)
	d, ok := tb.dirties[keyHex]
	if ok {
		if d.deleteFlag {
			return nil, ErrNotFound
		}
		return d.value, nil
	}
	return tb.Trie.Get(key)
}

// Put put to trie
func (tb *Batch) Put(key []byte, value []byte) error {
	tb.mu.Lock()
	if tb.batching {
		tb.batchPut(key, value)
		tb.mu.Unlock()
		return nil
	}
	tb.mu.Unlock()

	return tb.Trie.Put(key, value)
}

func (tb *Batch) batchPut(key []byte, value []byte) {
	keyHex := byteutils.Bytes2Hex(key)
	tb.dirties[keyHex] = &dirty{
		value:      value,
		deleteFlag: false,
	}
}

// GetData get value from trie and set on serializable data
func (tb *Batch) GetData(key []byte, data Serializable) error {
	value, err := tb.Get(key)
	if err != nil {
		return err
	}
	return data.FromBytes(value)
}

// PutData put serializable data to trie
func (tb *Batch) PutData(key []byte, data Serializable) error {
	value, err := data.ToBytes()
	if err != nil {
		return err
	}
	return tb.Put(key, value)
}

// RootHash getter for rootHash
func (tb *Batch) RootHash() ([]byte, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.batching {
		return nil, ErrCannotPerformInBatch
	}
	return tb.Trie.RootHash(), nil
}
