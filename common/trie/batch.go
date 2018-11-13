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
	batching bool
	dirties  map[string]*dirty
	mu       sync.RWMutex
}

// NewBatch return new Batch instance
func NewBatch(rootHash []byte, stor storage.Storage) (*Batch, error) {
	t, err := NewTrie(rootHash, stor)
	if err != nil {
		return nil, err
	}
	return &Batch{
		Trie:    t,
		dirties: make(map[string]*dirty),
	}, nil
}

// BeginBatch begin batch
func (tb *Batch) BeginBatch() error {
	if tb.batching {
		return ErrBeginAgainInBatch
	}
	tb.dirties = make(map[string]*dirty)
	tb.batching = true
	return nil
}

// Clone clone Batch
func (tb *Batch) Clone() (*Batch, error) {
	if tb.batching {
		return nil, ErrCannotCloneOnBatching
	}
	tr, err := tb.Trie.Clone()
	if err != nil {
		return nil, err
	}
	new := &Batch{
		Trie:    tr,
		dirties: make(map[string]*dirty),
	}
	return new, nil
}

// Commit commit batch WARNING: not thread-safe
func (tb *Batch) Commit() error {
	if !tb.batching {
		return ErrNotBatching
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	for keyHex, d := range tb.dirties {
		key := byteutils.Hex2Bytes(keyHex)
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

// Delete delete in trie
func (tb *Batch) Delete(key []byte) error {
	if !tb.batching {
		return ErrNotBatching
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
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
	if !tb.batching {
		return tb.Trie.Get(key)
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
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
	if !tb.batching {
		return ErrNotBatching
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()
	keyHex := byteutils.Bytes2Hex(key)
	tb.dirties[keyHex] = &dirty{
		value:      value,
		deleteFlag: false,
	}
	return nil
}

//GetData get value from trie and set on serializable data
func (tb *Batch) GetData(key []byte, data Serializable) error {
	value, err := tb.Get(key)
	if err != nil {
		return err
	}
	return data.FromBytes(value)
}

//PutData put serializable data to trie
func (tb *Batch) PutData(key []byte, data Serializable) error {
	value, err := data.ToBytes()
	if err != nil {
		return err
	}
	return tb.Put(key, value)
}

// RollBack rollback batch WARNING: not thread-safe
func (tb *Batch) RollBack() error {
	if !tb.batching {
		return ErrNotBatching
	}

	tb.dirties = make(map[string]*dirty)
	tb.batching = false
	return nil
}

// RootHash getter for rootHash
func (tb *Batch) RootHash() ([]byte, error) {
	if tb.batching {
		return nil, ErrNotBatching
	}
	return tb.Trie.RootHash(), nil
}
