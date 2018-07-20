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
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"encoding/hex"

	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// AccountStateBatch batch for AccountState
type AccountStateBatch struct {
	as            *AccountState
	batching      bool
	stageAccounts map[string]*Account
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
		as:            &AccountState{accounts: accTrie, storage: s},
		batching:      false,
		stageAccounts: make(map[string]*Account),
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

	for _, acc := range as.stageAccounts {
		bytes, err := acc.toBytes()
		if err != nil {
			return err
		}
		err = as.as.accounts.Put(acc.address, bytes)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Info("account put error")
		}
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

// GetAccount get account in stage(batching) or in original accountState
func (as *AccountStateBatch) GetAccount(address []byte) (*Account, error) {
	s := hex.EncodeToString(address)
	if acc, ok := as.stageAccounts[s]; ok {
		return acc, nil
	}
	stageAndReturn := func(acc2 *Account) *Account {
		as.stageAccounts[s] = acc2
		return acc2
	}
	accBytes, err := as.as.accounts.Get(address)
	if err != nil {
		// create account not exist in
		return stageAndReturn(&Account{
			address:       address,
			balance:       util.NewUint128(),
			vesting:       util.NewUint128(),
			voted:         []byte{},
			nonce:         0,
			records:       [][]byte{},
			certsReceived: [][]byte{},
			certsIssued:   [][]byte{},
		}), nil
	}
	acc, err := loadAccount(accBytes)
	if err != nil {
		return nil, err
	}
	return stageAndReturn(acc), nil
}

func (as *AccountStateBatch) resetBatch() {
	as.batching = false
	as.stageAccounts = make(map[string]*Account)
}

// AddBalance add balance
func (as *AccountStateBatch) AddBalance(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
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

// AddTransaction add transaction in account
func (as *AccountStateBatch) AddTxsFrom(address []byte, txHash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}

	acc.txsFrom = append(acc.txsFrom, txHash)
	return nil
}

func (as *AccountStateBatch) AddTxsTo(address []byte, txHash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}

	acc.txsTo = append(acc.txsTo, txHash)
	return nil
}

// AddRecord adds a record hash in account's records list
func (as *AccountStateBatch) AddRecord(address []byte, hash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
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

// AccountState getter for accountState
func (as *AccountStateBatch) AccountState() *AccountState {
	return as.as
}

// SubBalance subtract balance
func (as *AccountStateBatch) SubBalance(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
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
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}
	acc.nonce++
	return nil
}

// AddVesting increases vesting
func (as *AccountStateBatch) AddVesting(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}
	vesting, err := acc.vesting.Add(amount)
	if err != nil {
		return err
	}
	acc.vesting = vesting
	return nil
}

// SubVesting decreases vesting
func (as *AccountStateBatch) SubVesting(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}
	if amount.Cmp(acc.vesting) > 0 {
		return ErrVestingNotEnough
	}
	vesting, err := acc.vesting.Sub(amount)
	if err != nil {
		return err
	}
	acc.vesting = vesting
	return nil
}

// SetVoted vote sets voted of account
func (as *AccountStateBatch) SetVoted(address []byte, voted []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}
	if byteutils.Equal(acc.voted, voted) {
		return ErrAlreadyVoted
	}
	acc.voted = voted
	return nil
}

// GetVoted returned voted address of account
func (as *AccountStateBatch) GetVoted(address []byte) ([]byte, error) {
	acc, err := as.GetAccount(address)
	if err != nil {
		return nil, err
	}
	if len(acc.voted) == 0 {
		return nil, ErrNotVotedYet
	}
	return acc.voted, nil
}

// AddCertReceived adds a cert hash in certReceived
func (as *AccountStateBatch) AddCertReceived(address []byte, certHash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}
	for _, r := range acc.certsReceived {
		if byteutils.Equal(certHash, r) {
			return ErrCertReceivedAlreadyAdded
		}
	}
	acc.certsReceived = append(acc.certsReceived, certHash)
	return nil
}

// AddCertIssued adds a cert hash info in certIssued
func (as *AccountStateBatch) AddCertIssued(address []byte, certHash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.GetAccount(address)
	if err != nil {
		return err
	}
	for _, r := range acc.certsIssued {
		if byteutils.Equal(certHash, r) {
			return ErrCertIssuedAlreadyAdded
		}
	}
	acc.certsIssued = append(acc.certsIssued, certHash)
	return nil
}

// RootHash returns root hash of accounts trie
func (as *AccountStateBatch) RootHash() []byte {
	return as.as.accounts.RootHash()
}
