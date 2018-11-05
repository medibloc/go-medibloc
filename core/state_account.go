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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Prefixes for Account's Data trie
const (
	RecordsPrefix      = "r_"  // records
	CertReceivedPrefix = "cr_" // certs received
	CertIssuedPrefix   = "ci_" // certs issued
	AliasPrefix        = ""    // alias
)

// Account default item in state
type Account struct {
	Address common.Address // address account key
	Balance *util.Uint128  // balance account's coin amount
	Nonce   uint64         // nonce account sequential number
	Vesting *util.Uint128  // vesting account's vesting(staking) amount
	Voted   *trie.Batch    // voted candidates key: addr, value: addr

	Bandwidth       *util.Uint128
	LastBandwidthTs int64

	Unstaking       *util.Uint128
	LastUnstakingTs int64

	Collateral *util.Uint128 // candidate collateral
	Voters     *trie.Batch   // voters who voted to me
	VotePower  *util.Uint128 // sum of voters' vesting

	Data *trie.Batch // contain records, certReceived, certIssued

	Storage storage.Storage
}

func newAccount(stor storage.Storage) (*Account, error) {
	voted, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}
	voters, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}
	data, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	acc := &Account{
		Address:         common.Address{},
		Balance:         util.NewUint128(),
		Nonce:           0,
		Vesting:         util.NewUint128(),
		Voted:           voted,
		Bandwidth:       util.NewUint128(),
		LastBandwidthTs: 0,
		Unstaking:       util.NewUint128(),
		LastUnstakingTs: 0,
		Collateral:      util.NewUint128(),
		Voters:          voters,
		VotePower:       util.NewUint128(),
		Data:            data,
		Storage:         stor,
	}

	return acc, nil
}

func (acc *Account) fromProto(pbAcc *corepb.Account) error {
	var err error

	acc.Address = common.BytesToAddress(pbAcc.Address)
	acc.Balance, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Balance)
	if err != nil {
		return err
	}
	acc.Nonce = pbAcc.Nonce
	acc.Vesting, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Vesting)
	if err != nil {
		return err
	}
	acc.Voted, err = trie.NewBatch(pbAcc.VotedRootHash, acc.Storage)
	if err != nil {
		return err
	}

	acc.Bandwidth, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Bandwidth)
	if err != nil {
		return err
	}
	acc.LastBandwidthTs = pbAcc.LastBandwidthTs

	acc.Unstaking, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Unstaking)
	if err != nil {
		return err
	}
	acc.LastUnstakingTs = pbAcc.LastUnstakingTs

	acc.Collateral, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Collateral)
	if err != nil {
		return err
	}
	acc.Voters, err = trie.NewBatch(pbAcc.VotersRootHash, acc.Storage)
	if err != nil {
		return err
	}
	acc.VotePower, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.VotePower)
	if err != nil {
		return err
	}
	acc.Data, err = trie.NewBatch(pbAcc.DataRootHash, acc.Storage)
	if err != nil {
		return err
	}

	return nil
}

func (acc *Account) toProto() (*corepb.Account, error) {
	balance, err := acc.Balance.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	vesting, err := acc.Vesting.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	votedRootHash, err := acc.Voted.RootHash()
	if err != nil {
		return nil, err
	}
	bandwidth, err := acc.Bandwidth.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	unstaking, err := acc.Unstaking.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	collateral, err := acc.Collateral.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	votersRootHash, err := acc.Voters.RootHash()
	if err != nil {
		return nil, err
	}
	votePower, err := acc.VotePower.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	dataRootHash, err := acc.Data.RootHash()
	if err != nil {
		return nil, err
	}

	return &corepb.Account{
		Address:         acc.Address.Bytes(),
		Balance:         balance,
		Nonce:           acc.Nonce,
		Vesting:         vesting,
		VotedRootHash:   votedRootHash,
		Bandwidth:       bandwidth,
		LastBandwidthTs: acc.LastBandwidthTs,
		Unstaking:       unstaking,
		LastUnstakingTs: acc.LastUnstakingTs,
		Collateral:      collateral,
		VotersRootHash:  votersRootHash,
		VotePower:       votePower,
		DataRootHash:    dataRootHash,
	}, nil
}

func (acc *Account) toBytes() ([]byte, error) {
	pbAcc, err := acc.toProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pbAcc)
}

//VotedSlice returns slice converted from Voted trie
func (acc *Account) VotedSlice() [][]byte {
	return KeyTrieToSlice(acc.Voted)
}

//VotersSlice returns slice converted from Voters trie
func (acc *Account) VotersSlice() [][]byte {
	return KeyTrieToSlice(acc.Voters)
}

//GetData returns value in account's data trie
func (acc *Account) GetData(prefix string, key []byte) ([]byte, error) {
	return acc.Data.Get(append([]byte(prefix), key...))
}

//PutData put value to account's data trie
func (acc *Account) PutData(prefix string, key []byte, value []byte) error {
	return acc.Data.Put(append([]byte(prefix), key...), value)
}

//UpdateBandwidth update bandwidth
func (acc *Account) UpdateBandwidth(timestamp int64) error {
	var err error

	acc.Bandwidth, err = currentBandwidth(acc, timestamp)
	if err != nil {
		return err
	}
	acc.LastBandwidthTs = timestamp

	return nil
}

//UpdateUnstaking update unstaking and balance
func (acc *Account) UpdateUnstaking(timestamp int64) error {
	var err error
	if acc.LastUnstakingTs == 0 {
		return nil
	}

	elapsed := timestamp - acc.LastUnstakingTs
	if time.Duration(elapsed)*time.Second < UnstakingWaitDuration {
		return nil
	}

	acc.Balance, err = acc.Balance.Add(acc.Unstaking)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to add to balance.")
		return err
	}

	acc.Unstaking = util.NewUint128()
	acc.LastUnstakingTs = 0

	return nil
}

//AccountState is a struct for account state
type AccountState struct {
	*trie.Batch
	storage storage.Storage
}

//NewAccountState returns new AccountState
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

//Clone clones state
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

//GetAccount returns account
func (as *AccountState) GetAccount(addr common.Address) (*Account, error) {
	acc, err := newAccount(as.storage)
	if err != nil {
		return nil, err
	}
	accountBytes, err := as.Get(addr.Bytes())
	if err == ErrNotFound {
		acc.Address = addr
		return acc, nil
	}
	if err != nil {
		return nil, err
	}

	pbAccount := new(corepb.Account)
	if err := proto.Unmarshal(accountBytes, pbAccount); err != nil {
		return nil, err
	}
	if err := acc.fromProto(pbAccount); err != nil {
		return nil, err
	}

	return acc, nil
}

//putAccount put account to trie batch
func (as *AccountState) putAccount(acc *Account) error {
	accBytes, err := acc.toBytes()
	if err != nil {
		return err
	}
	return as.Put(acc.Address.Bytes(), accBytes)
}

// incrementNonce increment account's nonce
func (as *AccountState) incrementNonce(addr common.Address) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Nonce++
	return as.putAccount(acc)
}

//accounts returns account slice
func (as *AccountState) accounts() ([]*Account, error) {
	var accounts []*Account
	iter, err := as.Iterator(nil)
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
		acc, err := newAccount(as.storage)
		if err != nil {
			return nil, err
		}
		accountBytes := iter.Value()

		pbAccount := new(corepb.Account)
		if err := proto.Unmarshal(accountBytes, pbAccount); err != nil {
			return nil, err
		}
		if err := acc.fromProto(pbAccount); err != nil {
			return nil, err
		}

		accounts = append(accounts, acc)
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

//KeyTrieToSlice generate slice from trie (slice of key)
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

// AliasAccount alias for account
type AliasAccount struct {
	Account common.Address
}

func newAliasAccount() (*AliasAccount, error) {
	acc := &AliasAccount{
		Account: common.Address{},
	}
	return acc, nil
}

func (aa *AliasAccount) fromProto(pbAcc *corepb.AliasAccount) error {
	//var err error
	aa.Account = common.BytesToAddress(pbAcc.Account)
	return nil
}

func (aa *AliasAccount) toProto() (*corepb.AliasAccount, error) {
	return &corepb.AliasAccount{
		Account: aa.Account.Bytes(),
	}, nil
}

func (aa *AliasAccount) aliasAccountToBytes() ([]byte, error) {
	pbAcc, err := aa.toProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pbAcc)
}

//GetAliasAccount returns alias account
func (as *AccountState) GetAliasAccount(AliasName string) (*AliasAccount, error) {
	accountBytes, err := as.Get([]byte(AliasName))
	if err != nil {
		return nil, err
	}
	aa, err := newAliasAccount()
	if err != nil {
		return nil, err
	}
	pbAliasAccount := new(corepb.AliasAccount)
	if err := proto.Unmarshal(accountBytes, pbAliasAccount); err != nil {
		return nil, err
	}
	if err := aa.fromProto(pbAliasAccount); err != nil {
		return nil, err
	}

	return aa, nil
}

//PutAliasAccount put alias account to trie batch
func (as *AccountState) PutAliasAccount(acc *AliasAccount, aliasName string) error {
	aaBytes, err := acc.aliasAccountToBytes()
	if err != nil {
		return err
	}
	return as.Put([]byte(aliasName), aaBytes)
}
