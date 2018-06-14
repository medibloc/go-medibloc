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

package core

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// account default item in state
type account struct {
	// address account key
	address []byte
	// balance account's coin amount
	balance *util.Uint128
	// vesting account's vesting(staking) amount
	vesting *util.Uint128
	// nonce account sequential number
	nonce uint64
	// voted account
	voted []byte
	// records
	records [][]byte
	// certs received by a certifier
	certsReceived [][]byte
	// certs issued as a certifier
	certsIssued [][]byte
}

func (acc *account) Address() []byte {
	return acc.address
}

func (acc *account) Balance() *util.Uint128 {
	return acc.balance
}

func (acc *account) Vesting() *util.Uint128 {
	return acc.vesting
}

func (acc *account) Nonce() uint64 {
	return acc.nonce
}

func (acc *account) Voted() []byte {
	return acc.voted
}

func (acc *account) Records() [][]byte {
	return acc.records
}

func (acc *account) CertsReceived() [][]byte {
	return acc.certsReceived
}

func (acc *account) CertsIssued() [][]byte {
	return acc.certsIssued
}

func (acc *account) toBytes() ([]byte, error) {
	balanceBytes, err := acc.balance.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	vestingBytes, err := acc.vesting.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	pbAcc := &corepb.Account{
		Address:       acc.address,
		Balance:       balanceBytes,
		Vesting:       vestingBytes,
		Voted:         acc.voted,
		Nonce:         acc.nonce,
		Records:       acc.records,
		CertsReceived: acc.certsReceived,
		CertsIssued:   acc.certsIssued,
	}
	bytes, err := proto.Marshal(pbAcc)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func loadAccount(bytes []byte) (*account, error) {
	pbAcc := &corepb.Account{}
	if err := proto.Unmarshal(bytes, pbAcc); err != nil {
		return nil, err
	}
	balance := util.NewUint128()
	balance.FromFixedSizeByteSlice(pbAcc.Balance)
	vesting := util.NewUint128()
	vesting.FromFixedSizeByteSlice(pbAcc.Vesting)
	acc := &account{
		address:       pbAcc.Address,
		balance:       balance,
		vesting:       vesting,
		voted:         pbAcc.Voted,
		nonce:         pbAcc.Nonce,
		records:       pbAcc.Records,
		certsReceived: pbAcc.CertsReceived,
		certsIssued:   pbAcc.CertsIssued,
	}
	return acc, nil
}

// accountState
type accountState struct {
	accounts *trie.Trie
	storage  storage.Storage
}

// GetAccount get account for address
func (as *accountState) GetAccount(address []byte) (Account, error) {
	bytes, err := as.accounts.Get(address)
	if err != nil {
		return nil, err
	}
	return loadAccount(bytes)
}

func (as *accountState) Accounts() ([]Account, error) {
	var accounts []Account
	iter, err := as.accounts.Iterator(nil)
	if err == trie.ErrNotFound {
		return accounts, nil
	}
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get iterator of account trie.")
		return nil, err
	}

	exist, err := iter.Next()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to iterate account trie.")
		return nil, err
	}

	for exist {
		accBytes := iter.Value()
		account, err := loadAccount(accBytes)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":   err,
				"bytes": accBytes,
			}).Error("Failed to get accounts from trie.")
			return nil, err
		}
		accounts = append(accounts, account)
		exist, err = iter.Next()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to iterate account trie.")
			return nil, err
		}
	}
	return accounts, nil

}

// Account account interface
type Account interface {
	// Address getter for address
	Address() []byte

	// Balance getter for balance
	Balance() *util.Uint128

	Vesting() *util.Uint128

	Nonce() uint64

	Voted() []byte

	Records() [][]byte

	CertsReceived() [][]byte

	CertsIssued() [][]byte
}

// AccountState account state interface
type AccountState interface {
	// GetAccount get account for address
	GetAccount(address []byte) (Account, error)

	Accounts() ([]Account, error)
}
