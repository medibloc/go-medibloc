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

package corestate

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Prefixes for blockState trie
const (
	AccountPrefix      = "ac_" // alias account prefix for account state trie
	AliasAccountPrefix = "al_" // alias account prefix for account state trie
)

// AccountState is a struct for account state
type AccountState struct {
	*trie.Batch
	storage storage.Storage
}

// NewAccountState returns new AccountState
func NewAccountState(rootHash []byte, stor storage.Storage) (*AccountState, error) {
	trieBatch, err := trie.NewBatch(rootHash, stor)
	if err != nil {
		return nil, err
	}

	return &AccountState{
		Batch:   trieBatch,
		storage: stor,
	}, nil
}

// Clone clones state
func (as *AccountState) Clone() (*AccountState, error) {
	newBatch, err := as.Batch.Clone()
	if err != nil {
		return nil, err
	}
	return &AccountState{
		Batch:   newBatch,
		storage: as.storage,
	}, nil
}

// GetAccount returns account
func (as *AccountState) GetAccount(addr common.Address, timestamp int64) (*Account, error) {
	acc, err := newAccount(as.storage)
	if err != nil {
		return nil, err
	}
	err = as.GetData(append([]byte(AccountPrefix), addr.Bytes()...), acc)
	if err == ErrNotFound {
		acc.Address = addr
		return acc, nil
	}
	if err != nil {
		return nil, err
	}

	if err := acc.updatePoints(timestamp); err != nil {
		return nil, err
	}

	if err := acc.updateUnstaking(timestamp); err != nil {
		return nil, err
	}

	return acc, nil
}

// PutAccount put account to trie batch
func (as *AccountState) PutAccount(acc *Account) error {
	return as.PutData(append([]byte(AccountPrefix), acc.Address.Bytes()...), acc)
}

// Accounts returns account slice, except alias account
func (as *AccountState) Accounts() ([]*Account, error) {
	var accounts []*Account
	iter, err := as.Iterator([]byte(AccountPrefix))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get iterator of account trie.")
		return nil, err
	}

	for {
		exist, err := iter.Next()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to iterate account trie.")
			return nil, err
		}
		if !exist {
			break
		}

		accountBytes := iter.Value()

		acc, err := newAccount(as.storage)
		if err != nil {
			return nil, err
		}
		if err := acc.FromBytes(accountBytes); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

// KeyTrieToSlice generate slice from trie (slice of key)
func KeyTrieToSlice(trie *trie.Batch) [][]byte {
	var slice [][]byte
	iter, err := trie.Iterator(nil)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get iterator of trie.")
		return nil
	}

	exist, err := iter.Next()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to iterate account trie.")
		return nil
	}

	for exist {
		slice = append(slice, iter.Key())

		exist, err = iter.Next()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to iterate account trie.")
			return nil
		}
	}
	return slice
}

// GetAccountByAlias returns account by alias
func (as *AccountState) GetAccountByAlias(alias string, timestamp int64) (*Account, error) {
	addrBytes, err := as.Get(append([]byte(AliasAccountPrefix), []byte(alias)...))
	if err != nil {
		return nil, err
	}
	addr, err := common.BytesToAddress(addrBytes)
	if err != nil {
		return nil, err
	}
	return as.GetAccount(addr, timestamp)
}

// PutAccountAlias put alias name for address
func (as *AccountState) PutAccountAlias(alias string, addr common.Address) error {
	return as.Put(append([]byte(AliasAccountPrefix), []byte(alias)...), addr.Bytes())

}

// DelAccountAlias del alias name info
func (as *AccountState) DelAccountAlias(alias string, addr common.Address) error {
	addrBytes, err := as.Get(append([]byte(AliasAccountPrefix), []byte(alias)...))
	if err != nil {
		return err
	}
	owner, err := common.BytesToAddress(addrBytes)
	if err != nil {
		return err
	}
	if !owner.Equals(addr) {
		return ErrUnauthorized
	}

	return as.Delete(append([]byte(AliasAccountPrefix), []byte(alias)...))
}
