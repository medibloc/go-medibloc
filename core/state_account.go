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
	corepb "github.com/medibloc/go-medibloc/core/pb"
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
	AliasPrefix        = ""    // alias prefix for data trie
	AccountPrefix      = "ac_" // alias account prefix for account state trie
	AliasAccountPrefix = "al_" // alias account prefix for account state trie
)

// Account default item in state
type Account struct {
	Address     common.Address // address account key
	Balance     *util.Uint128  // balance account's coin amount
	Nonce       uint64         // nonce account sequential number
	Staking     *util.Uint128  // account's staking amount
	Voted       *trie.Batch    // voted candidates key: candidateID, value: candidateID
	CandidateID []byte         //

	Points       *util.Uint128
	LastPointsTs int64

	Unstaking       *util.Uint128
	LastUnstakingTs int64

	Data *trie.Batch // contain records, certReceived, certIssued

	Storage storage.Storage
}

func newAccount(stor storage.Storage) (*Account, error) {
	voted, err := trie.NewBatch(nil, stor)
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
		Staking:         util.NewUint128(),
		Voted:           voted,
		CandidateID:     nil,
		Points:          util.NewUint128(),
		LastPointsTs:    0,
		Unstaking:       util.NewUint128(),
		LastUnstakingTs: 0,
		Data:            data,
		Storage:         stor,
	}

	return acc, nil
}

func (acc *Account) fromProto(pbAcc *corepb.Account) error {
	var err error

	acc.Address.FromBytes(pbAcc.Address)
	acc.Balance, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Balance)
	if err != nil {
		return err
	}
	acc.Nonce = pbAcc.Nonce
	acc.Staking, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Staking)
	if err != nil {
		return err
	}
	acc.Voted, err = trie.NewBatch(pbAcc.VotedRootHash, acc.Storage)
	if err != nil {
		return err
	}

	acc.CandidateID = pbAcc.CandidateId

	acc.Points, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Bandwidth)
	if err != nil {
		return err
	}
	acc.LastPointsTs = pbAcc.LastBandwidthTs

	acc.Unstaking, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Unstaking)
	if err != nil {
		return err
	}
	acc.LastUnstakingTs = pbAcc.LastUnstakingTs

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
	staking, err := acc.Staking.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	votedRootHash, err := acc.Voted.RootHash()
	if err != nil {
		return nil, err
	}
	bandwidth, err := acc.Points.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	unstaking, err := acc.Unstaking.ToFixedSizeByteSlice()
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
		Staking:         staking,
		VotedRootHash:   votedRootHash,
		CandidateId:     acc.CandidateID,
		Bandwidth:       bandwidth,
		LastBandwidthTs: acc.LastPointsTs,
		Unstaking:       unstaking,
		LastUnstakingTs: acc.LastUnstakingTs,
		DataRootHash:    dataRootHash,
	}, nil
}

// FromBytes returns Account form bytes
func (acc *Account) FromBytes(b []byte) error {
	pbAccount := new(corepb.Account)
	if err := proto.Unmarshal(b, pbAccount); err != nil {
		return err
	}
	return acc.fromProto(pbAccount)
}

// ToBytes convert account to bytes
func (acc *Account) ToBytes() ([]byte, error) {
	pbAcc, err := acc.toProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pbAcc)
}

// VotedSlice returns slice converted from Voted trie
func (acc *Account) VotedSlice() [][]byte {
	return KeyTrieToSlice(acc.Voted)
}

// GetData returns value in account's data trie
func (acc *Account) GetData(prefix string, key []byte) ([]byte, error) {
	return acc.Data.Get(append([]byte(prefix), key...))
}

// PutData put value to account's data trie
func (acc *Account) PutData(prefix string, key []byte, value []byte) error {
	return acc.Data.Put(append([]byte(prefix), key...), value)
}

// UpdatePoints update points
func (acc *Account) UpdatePoints(timestamp int64) error {
	var err error

	acc.Points, err = currentPoints(acc, timestamp)
	if err != nil {
		return err
	}
	acc.LastPointsTs = timestamp

	return nil
}

// UpdateUnstaking update unstaking and balance
func (acc *Account) UpdateUnstaking(timestamp int64) error {
	var err error
	// Unstaking action does not exist
	if acc.LastUnstakingTs == 0 {
		return nil
	}

	// Staked coin is not returned if not enough time has been passed
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

// checkAccountPoints compare given transaction's required bandwidth with the account's remaining bandwidth
func (acc *Account) checkAccountPoints(transaction *Transaction, points *util.Uint128) error {
	var err error
	avail := acc.Points
	switch transaction.TxType() {
	case TxOpStake:
		avail, err = acc.Points.Add(transaction.Value())
		if err != nil {
			return err
		}
	case TxOpUnstake:
		avail, err = acc.Points.Sub(transaction.Value())
		if err == util.ErrUint128Underflow {
			return ErrStakingNotEnough
		}
		if err != nil {
			return err
		}
	}
	if avail.Cmp(points) < 0 {
		return ErrPointNotEnough
	}
	return nil
}

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
func (as *AccountState) GetAccount(addr common.Address) (*Account, error) {
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

	return acc, nil
}

// putAccount put account to trie batch
func (as *AccountState) putAccount(acc *Account) error {
	return as.PutData(append([]byte(AccountPrefix), acc.Address.Bytes()...), acc)
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

// accounts returns account slice, except alias account
func (as *AccountState) accounts() ([]*Account, error) {
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

// AliasAccount alias for account
type AliasAccount struct {
	Account common.Address
	Alias   string
}

func newAliasAccount() (*AliasAccount, error) {
	acc := &AliasAccount{
		Account: common.Address{},
		Alias:   "",
	}
	return acc, nil
}

func (aa *AliasAccount) fromProto(pbAcc *corepb.AliasAccount) error {
	aa.Account.FromBytes(pbAcc.Account)
	aa.Alias = pbAcc.Alias
	return nil
}

func (aa *AliasAccount) toProto() (*corepb.AliasAccount, error) {
	return &corepb.AliasAccount{
		Account: aa.Account.Bytes(),
		Alias:   aa.Alias,
	}, nil
}

func (aa *AliasAccount) aliasAccountToBytes() ([]byte, error) {
	pbAcc, err := aa.toProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pbAcc)
}

// GetAliasAccount returns alias account
func (as *AccountState) GetAliasAccount(AliasName string) (*AliasAccount, error) {
	accountBytes, err := as.Get(append([]byte(AliasAccountPrefix), []byte(AliasName)...))
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

// PutAliasAccount put alias account to trie batch
func (as *AccountState) PutAliasAccount(acc *AliasAccount, aliasName string) error {
	aaBytes, err := acc.aliasAccountToBytes()
	if err != nil {
		return err
	}
	return as.Put(append([]byte(AliasAccountPrefix), []byte(aliasName)...), aaBytes)
}
