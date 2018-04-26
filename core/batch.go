// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
)

// Errors
var (
	ErrNotFound = storage.ErrKeyNotFound
)

// Action represents operation types in BatchTrie
type Action int

// Action constants
const (
	Insert Action = iota
	Update
	Delete
)

// Batch batch interface
type Batch interface {
	BeginBatch() error
	Commit() error
	RollBack() error
}

// Entry entry for not committed changes
type Entry struct {
	action Action
	key    []byte
	old    []byte
	update []byte
}

// TrieBatch batch implementation for trie
type TrieBatch struct {
	batching    bool
	changeLogs  []*Entry
	oldRootHash []byte
	trie        *trie.Trie
	storage     storage.Storage
}

// NewTrieBatch return new TrieBatch instance
func NewTrieBatch(rootHash []byte, storage storage.Storage) (*TrieBatch, error) {
	t, err := trie.NewTrie(rootHash, storage)
	if err != nil {
		return nil, err
	}
	return &TrieBatch{
		trie:    t,
		storage: storage,
	}, nil
}

// BeginBatch begin batch
func (t *TrieBatch) BeginBatch() error {
	if t.batching {
		return ErrBeginAgainInBatch
	}
	t.oldRootHash = t.trie.RootHash()
	t.batching = true
	return nil
}

// Clone clone TrieBatch
func (t *TrieBatch) Clone() (*TrieBatch, error) {
	if t.batching {
		return nil, ErrCannotCloneOnBatching
	}
	return NewTrieBatch(t.trie.RootHash(), t.storage)
}

// Commit commit batch WARNING: not thread-safe
func (t *TrieBatch) Commit() error {
	if !t.batching {
		return ErrNotBatching
	}
	t.changeLogs = t.changeLogs[:0]
	t.batching = false
	return nil
}

// Delete delete in trie
func (t *TrieBatch) Delete(key []byte) error {
	if !t.batching {
		return ErrNotBatching
	}
	entry := &Entry{action: Delete, key: key, old: nil}
	old, getErr := t.trie.Get(key)
	if getErr == nil {
		entry.old = old
	}
	t.changeLogs = append(t.changeLogs, entry)
	return t.trie.Delete(key)
}

// Get get from trie
func (t *TrieBatch) Get(key []byte) ([]byte, error) {
	return t.trie.Get(key)
}

// Iterator iterates trie.
func (t *TrieBatch) Iterator(prefix []byte) (*trie.Iterator, error) {
	return t.trie.Iterator(prefix)
}

// Put put to trie
func (t *TrieBatch) Put(key []byte, value []byte) error {
	if !t.batching {
		return ErrNotBatching
	}
	entry := &Entry{action: Insert, key: key, old: nil, update: value}
	old, getErr := t.trie.Get(key)
	if getErr == nil {
		entry.action = Update
		entry.old = old
	}
	t.changeLogs = append(t.changeLogs, entry)
	return t.trie.Put(key, value)
}

// RollBack rollback batch WARNING: not thread-safe
func (t *TrieBatch) RollBack() error {
	if !t.batching {
		return ErrNotBatching
	}
	// TODO rollback with changelogs
	t.changeLogs = t.changeLogs[:0]
	t.trie.SetRootHash(t.oldRootHash)
	t.batching = false
	return nil
}

// RootHash getter for rootHash
func (t *TrieBatch) RootHash() []byte {
	return t.trie.RootHash()
}
