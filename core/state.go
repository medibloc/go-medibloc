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
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
)

// account default item in state
type account struct {
	// address account key
	address []byte
	// balance account's coin amount
	balance *util.Uint128
	// nonce account sequential number
	nonce uint64
	// observations
	observations *TrieBatch
}

func (acc *account) Address() []byte {
	return acc.address
}

func (acc *account) Balance() *util.Uint128 {
	return acc.balance
}

func (acc *account) toBytes() ([]byte, error) {
	bytes, err := acc.balance.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	pbAcc := &corepb.Account{
		Address:          acc.address,
		Balance:          bytes,
		Nonce:            acc.nonce,
		ObservationsHash: acc.observations.RootHash(),
	}
	bytes, err = proto.Marshal(pbAcc)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func loadAccount(bytes []byte, storage storage.Storage) (*account, error) {
	pbAcc := &corepb.Account{}
	if err := proto.Unmarshal(bytes, pbAcc); err != nil {
		return nil, err
	}
	balance := util.NewUint128()
	balance.FromFixedSizeByteSlice(pbAcc.Balance)
	acc := &account{
		address: pbAcc.Address,
		balance: balance,
		nonce:   pbAcc.Nonce,
	}
	t, err := trie.NewTrie(pbAcc.ObservationsHash, storage)
	if err != nil {
		return nil, err
	}
	acc.observations = &TrieBatch{trie: t}
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
	return loadAccount(bytes, as.storage)
}

// Account account interface
type Account interface {
	// Address getter for address
	Address() []byte

	// Balance getter for balance
	Balance() *util.Uint128
}

// AccountState account state interface
type AccountState interface {
	// GetAccount get account for address
	GetAccount(address []byte) (Account, error)
}
