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
	// vesting account's vesting(staking) amount
	vesting *util.Uint128
	// nonce account sequential number
	nonce uint64
	// voted account
	voted []byte
	// writers
	writers [][]byte
	// records
	records [][]byte
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

func (acc *account) Writers() [][]byte {
	return acc.writers
}

func (acc *account) Records() [][]byte {
	return acc.records
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
		Address: acc.address,
		Balance: balanceBytes,
		Vesting: vestingBytes,
		Voted:   acc.voted,
		Nonce:   acc.nonce,
		Writers: acc.writers,
		Records: acc.records,
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
		address: pbAcc.Address,
		balance: balance,
		vesting: vesting,
		voted:   pbAcc.Voted,
		nonce:   pbAcc.Nonce,
		writers: pbAcc.Writers,
		records: pbAcc.Records,
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

// Account account interface
type Account interface {
	// Address getter for address
	Address() []byte

	// Balance getter for balance
	Balance() *util.Uint128

	Vesting() *util.Uint128

	Nonce() uint64

	Voted() []byte

	Writers() [][]byte

	Records() [][]byte
}

// AccountState account state interface
type AccountState interface {
	// GetAccount get account for address
	GetAccount(address []byte) (Account, error)
}
