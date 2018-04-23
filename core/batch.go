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
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// Errors
var (
	ErrBalanceNotEnough      = errors.New("remaining balance is not enough")
	ErrBeginAgainInBatch     = errors.New("cannot begin with a batch task unfinished")
	ErrCannotCloneOnBatching = errors.New("cannot clone on batching")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrNotBatching           = errors.New("not batching")
	ErrNotFound              = storage.ErrKeyNotFound
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

// AccountStateBatch batch for AccountState
type AccountStateBatch struct {
	as            *accountState
	batching      bool
	stageAccounts map[string]*account
	storage       storage.Storage
}

// NewAccountStateBatch create and return new AccountStateBatch instance
func NewAccountStateBatch(accountsRootHash []byte, s storage.Storage) (*AccountStateBatch, error) {
	// TODO Get initial accounts from config or another place
	accTrie, err := trie.NewTrie(accountsRootHash, s)
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

// Clone clone AccountStateBatch enable only when not batching
func (as *AccountStateBatch) Clone() (*AccountStateBatch, error) {
	if as.batching {
		return nil, ErrCannotCloneOnBatching
	}
	return NewAccountStateBatch(as.as.accounts.RootHash(), as.storage)
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

	as.resetBatch()
	return nil
}

// RollBack rollback batch WARNING: not thread-safe
func (as *AccountStateBatch) RollBack() error {
	if !as.batching {
		return ErrNotBatching
	}
	as.resetBatch()
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
		return stageAndReturn(&account{
			address: address,
			balance: util.NewUint128(),
			nonce:   0,
		}), nil
	}
	acc, err := loadAccount(accBytes, as.storage)
	if err != nil {
		return nil, err
	}
	return stageAndReturn(acc), nil
}

func (as *AccountStateBatch) resetBatch() {
	as.batching = false
	as.stageAccounts = make(map[string]*account)
}

// AddBalance add balance
func (as *AccountStateBatch) AddBalance(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	balance, err := acc.balance.Add(amount)
	if err != nil {
		return err
	}
	acc.balance = balance
	return nil
}

// AddWriter adds writer address of this account
// writers can use the account's bandwidth and add record of her
func (as *AccountStateBatch) AddWriter(address []byte, newWriter []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	for _, w := range acc.writers {
		if byteutils.Equal(newWriter, w) {
			return ErrWriterAlreadyRegistered
		}
	}
	acc.writers = append(acc.writers, newWriter)
	return nil
}

// RemoveWriter removes a writer from account's writers list
func (as *AccountStateBatch) RemoveWriter(address []byte, writer []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	writers := [][]byte{}
	for _, w := range acc.writers {
		if byteutils.Equal(writer, w) {
			continue
		}
		writers = append(writers, w)
	}
	if len(acc.writers) == len(writers) {
		return ErrWriterNotFound
	}
	acc.writers = writers
	return nil
}

// AddRecord adds a record hash in account's records list
func (as *AccountStateBatch) AddRecord(address []byte, hash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	for _, r := range acc.records {
		if byteutils.Equal(hash, r) {
			return ErrRecordAlreadyAdded
		}
	}
	acc.records = append(acc.records, hash)
	return nil
}

// GetAccount get account in stage(batching) or in original accountState
func (as *AccountStateBatch) GetAccount(address []byte) (Account, error) {
	s := hex.EncodeToString(address)
	if acc, ok := as.stageAccounts[s]; ok {
		return acc, nil
	}
	accBytes, err := as.as.accounts.Get(address)
	if err == nil {
		acc, err := loadAccount(accBytes, as.storage)
		if err != nil {
			return nil, err
		}
		return acc, nil
	}
	return nil, ErrNotFound
}

// AccountState getter for accountState
func (as *AccountStateBatch) AccountState() AccountState {
	return as.as
}

// SubBalance subtract balance
func (as *AccountStateBatch) SubBalance(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	if amount.Cmp(acc.balance) > 0 {
		return ErrBalanceNotEnough
	}
	balance, err := acc.balance.Sub(amount)
	if err != nil {
		return err
	}
	acc.balance = balance
	return nil
}

// IncrementNonce increment account's nonce
func (as *AccountStateBatch) IncrementNonce(address []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	acc.nonce++
	return nil
}

// RootHash returns root hash of accounts trie
func (as *AccountStateBatch) RootHash() []byte {
	return as.as.accounts.RootHash()
}
