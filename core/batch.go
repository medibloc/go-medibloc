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
	"encoding/hex"
	"errors"

	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
)

// Errors
var (
	ErrBalanceNotEnough  = errors.New("remaining balance is not enough")
	ErrBeginAgainInBatch = errors.New("cannot begin with a batch task unfinished")
	ErrNotFound          = storage.ErrKeyNotFound
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrNotBatching       = errors.New("not batching")
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
}

// NewTrieBatch return new TrieBatch instance
func NewTrieBatch(rootHash []byte, storage storage.Storage) (*TrieBatch, error) {
	t, err := trie.NewTrie(rootHash, storage)
	if err != nil {
		return nil, err
	}
	return &TrieBatch{
		trie: t,
	}, nil
}

// Get get from trie
func (t *TrieBatch) Get(key []byte) ([]byte, error) {
	return t.trie.Get(key)
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

// RootHash getter for rootHash
func (t *TrieBatch) RootHash() []byte {
	return t.trie.RootHash()
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

// Commit commit batch WARNING: not thread-safe
func (t *TrieBatch) Commit() error {
	if !t.batching {
		return ErrNotBatching
	}
	t.changeLogs = t.changeLogs[:0]
	t.batching = false
	return nil
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

// AccountStateBatch batch for AccountState
type AccountStateBatch struct {
	as            *accountState
	batching      bool
	stageAccounts map[string]*account
	storage       storage.Storage
}

// NewAccountStateBatch create and return new AccountStateBatch instance
func NewAccountStateBatch(s storage.Storage) (*AccountStateBatch, error) {
	// TODO Get initial accounts from config or another place
	accTrie, err := trie.NewTrie(nil, s)
	if err != nil {
		return nil, err
	}
	return &AccountStateBatch{
		as:            &accountState{accounts: accTrie, storage: s},
		batching:      false,
		stageAccounts: make(map[string]*account),
		storage:       s,
	}, nil
}

// BeginBatch begin batch
func (as *AccountStateBatch) BeginBatch() error {
	if as.batching {
		return ErrBeginAgainInBatch
	}
	as.batching = true
	return nil
}

// Commit commit batch WARNING: not thread-safe
func (as *AccountStateBatch) Commit() error {
	if !as.batching {
		return ErrNotBatching
	}

	for k, acc := range as.stageAccounts {
		bytes, err := hex.DecodeString(k)
		if err != nil {
			return err
		}
		bytes, err = acc.toBytes()
		if err != nil {
			return err
		}
		as.as.accounts.Put(acc.address, bytes)
	}
	as.batching = false
	return nil
}

// RollBack rollback batch WARNING: not thread-safe
func (as *AccountStateBatch) RollBack() error {
	if !as.batching {
		return ErrNotBatching
	}
	as.batching = false
	as.stageAccounts = make(map[string]*account)
	return nil
}

func (as *AccountStateBatch) getAccount(address []byte) (*account, error) {
	s := hex.EncodeToString(address)
	if acc, ok := as.stageAccounts[s]; ok {
		return acc, nil
	}
	stageAndReturn := func(acc2 *account) *account {
		as.stageAccounts[s] = acc2
		return acc2
	}
	accBytes, err := as.as.accounts.Get(address)
	if err != nil {
		// create account not exist in
		observations, err := trie.NewTrie(nil, as.storage)
		if err != nil {
			return nil, err
		}
		return stageAndReturn(&account{
			address:      address,
			balance:      0,
			nonce:        1,
			observations: &TrieBatch{trie: observations},
		}), nil
	}
	acc, err := loadAccount(accBytes, as.storage)
	if err != nil {
		return nil, err
	}
	return stageAndReturn(acc), nil
}

// AddBalance add balance
func (as *AccountStateBatch) AddBalance(address []byte, amount uint64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	acc.balance += amount
	return nil
}

// AddObservation add observation
func (as *AccountStateBatch) AddObservation(address []byte, hash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	acc.observations.Put(hash, hash) // TODO
	return nil
}

// GetAccount get account in stage(batching)
func (as *AccountStateBatch) GetAccount(address []byte) (*account, error) {
	if !as.batching {
		return nil, ErrNotBatching
	}
	s := hex.EncodeToString(address)
	if acc, ok := as.stageAccounts[s]; ok {
		return acc, nil
	}
	return nil, ErrNotFound
}

// AccountState getter for accountState
func (as *AccountStateBatch) AccountState() *accountState {
	return as.as
}

// SubBalance subtract balance
func (as *AccountStateBatch) SubBalance(address []byte, amount uint64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	if amount > acc.balance {
		return ErrBalanceNotEnough
	}
	acc.balance -= amount
	return nil
}
