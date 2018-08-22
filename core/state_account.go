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
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Account default item in state
type Account struct {
	Address common.Address // address account key
	Balance *util.Uint128  // balance account's coin amount
	Nonce   uint64         // nonce account sequential number
	Vesting *util.Uint128  // vesting account's vesting(staking) amount
	Voted   *trie.Trie     // voted candidates key: addr, value: addr

	Collateral *util.Uint128 // candidate collateral
	Voters     *trie.Trie    // voters who voted to me
	VotePower  *util.Uint128 // sum of voters' vesting

	Records       *trie.Trie // records
	CertsReceived *trie.Trie // certs received by a certifier
	CertsIssued   *trie.Trie // certs issued as a certifier
	TxsFrom       *trie.Trie // transaction sent from account
	TxsTo         *trie.Trie // transaction sent to account

	Storage storage.Storage
}

func newAccount(stor storage.Storage) (*Account, error) {
	voted, err := trie.NewTrie(nil, stor)
	if err != nil {
		return nil, err
	}
	voters, err := trie.NewTrie(nil, stor)
	if err != nil {
		return nil, err
	}
	records, err := trie.NewTrie(nil, stor)
	if err != nil {
		return nil, err
	}
	certReceived, err := trie.NewTrie(nil, stor)
	if err != nil {
		return nil, err
	}
	certIssued, err := trie.NewTrie(nil, stor)
	if err != nil {
		return nil, err
	}
	TxsFrom, err := trie.NewTrie(nil, stor)
	if err != nil {
		return nil, err
	}
	TxsTo, err := trie.NewTrie(nil, stor)
	if err != nil {
		return nil, err
	}

	acc := &Account{
		Address:       common.Address{},
		Balance:       util.NewUint128(),
		Nonce:         0,
		Vesting:       util.NewUint128(),
		Voted:         voted,
		Collateral:    util.NewUint128(),
		Voters:        voters,
		VotePower:     util.NewUint128(),
		Records:       records,
		CertsReceived: certReceived,
		CertsIssued:   certIssued,
		TxsFrom:       TxsFrom,
		TxsTo:         TxsTo,
		Storage:       stor,
	}

	return acc, nil
}

func (acc *Account) fromProto(pbAcc *corepb.AccountState) error {
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
	acc.Voted.SetRootHash(pbAcc.VotedRootHash)
	acc.Collateral, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.Collateral)
	if err != nil {
		return err
	}
	acc.Voters.SetRootHash(pbAcc.VotersRootHash)
	acc.VotePower, err = util.NewUint128FromFixedSizeByteSlice(pbAcc.VotePower)
	if err != nil {
		return err
	}
	acc.Records.SetRootHash(pbAcc.RecordsRootHash)
	acc.CertsReceived.SetRootHash(pbAcc.CertsReceivedRootHash)
	acc.CertsIssued.SetRootHash(pbAcc.CertsIssuedRootHash)
	acc.TxsFrom.SetRootHash(pbAcc.TxsFromRootHash)
	acc.TxsTo.SetRootHash(pbAcc.TxsToRootHash)

	return nil
}

func (acc *Account) fromBytes(accountBytes []byte) error {
	pbAccount := new(corepb.AccountState)
	if err := proto.Unmarshal(accountBytes, pbAccount); err != nil {
		return err
	}
	if err := acc.fromProto(pbAccount); err != nil {
		return err
	}
	return nil
}

//LoadAccount returns account from accountBytes and storage
func LoadAccount(accountBytes []byte, stor storage.Storage) (*Account, error) {
	acc, err := newAccount(stor)
	if err != nil {
		return nil, err
	}
	if err := acc.fromBytes(accountBytes); err != nil {
		return nil, err
	}
	return acc, nil
}

func (acc *Account) toProto() (*corepb.AccountState, error) {
	balance, err := acc.Balance.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	vesting, err := acc.Vesting.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	collateral, err := acc.Collateral.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	votePower, err := acc.VotePower.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	return &corepb.AccountState{
		Address:               acc.Address.Bytes(),
		Balance:               balance,
		Nonce:                 acc.Nonce,
		Vesting:               vesting,
		VotedRootHash:         acc.Voted.RootHash(),
		Collateral:            collateral,
		VotersRootHash:        acc.Voters.RootHash(),
		VotePower:             votePower,
		RecordsRootHash:       acc.Records.RootHash(),
		CertsReceivedRootHash: acc.CertsReceived.RootHash(),
		CertsIssuedRootHash:   acc.CertsIssued.RootHash(),
		TxsFromRootHash:       acc.TxsFrom.RootHash(),
		TxsToRootHash:         acc.TxsTo.RootHash(),
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

//RecordsSlice returns slice converted from Records trie
func (acc *Account) RecordsSlice() [][]byte {
	return KeyTrieToSlice(acc.Records)
}

//CertsReceivedSlice returns slice converted from CertsReceived trie
func (acc *Account) CertsReceivedSlice() [][]byte {
	return KeyTrieToSlice(acc.CertsReceived)
}

//CertsIssuedSlice returns slice converted from CertsIssued trie
func (acc *Account) CertsIssuedSlice() [][]byte {
	return KeyTrieToSlice(acc.CertsIssued)
}

//TxsFromSlice returns slice converted from TxsFrom trie
func (acc *Account) TxsFromSlice() [][]byte {
	return KeyTrieToSlice(acc.TxsFrom)
}

//TxsToSlice returns slice converted from TxsTo trie
func (acc *Account) TxsToSlice() [][]byte {
	return KeyTrieToSlice(acc.TxsTo)
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

//GetAccount returns account
func (as *AccountState) GetAccount(addr common.Address) (*Account, error) {
	accountBytes, err := as.Get(addr.Bytes())
	if err == ErrNotFound {
		acc, err := newAccount(as.storage)
		if err != nil {
			return nil, err
		}
		acc.Address = addr
		return acc, nil
	}
	if err != nil {
		return nil, err
	}

	return LoadAccount(accountBytes, as.storage)
}

//PutAccount put account to trie batch
func (as *AccountState) PutAccount(acc *Account) error {
	accBytes, err := acc.toBytes()
	if err != nil {
		return err
	}
	return as.Put(acc.Address.Bytes(), accBytes)
}

// IncrementNonce increment account's nonce
func (as *AccountState) IncrementNonce(addr common.Address) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Nonce++
	return as.PutAccount(acc)
}

// AddBalance add balance
func (as *AccountState) AddBalance(addr common.Address, amount *util.Uint128) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	balance, err := acc.Balance.Add(amount)
	if err != nil {
		return err
	}
	acc.Balance = balance

	return as.PutAccount(acc)
}

// SubBalance subtract balance
func (as *AccountState) SubBalance(addr common.Address, amount *util.Uint128) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	if amount.Cmp(acc.Balance) > 0 {
		return ErrBalanceNotEnough
	}
	balance, err := acc.Balance.Sub(amount)
	if err != nil {
		return err
	}
	acc.Balance = balance
	return as.PutAccount(acc)
}

// AddVesting increases vesting
func (as *AccountState) AddVesting(addr common.Address, amount *util.Uint128) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	vesting, err := acc.Vesting.Add(amount)
	if err != nil {
		return err
	}
	acc.Vesting = vesting
	return as.PutAccount(acc)
}

// SubVesting decreases vesting
func (as *AccountState) SubVesting(addr common.Address, amount *util.Uint128) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	if amount.Cmp(acc.Vesting) > 0 {
		return ErrVestingNotEnough
	}
	vesting, err := acc.Vesting.Sub(amount)
	if err != nil {
		return err
	}
	acc.Vesting = vesting
	return as.PutAccount(acc)
}

//SetVoted sets voted trie
func (as *AccountState) SetVoted(addr common.Address, candidates []common.Address) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}

	acc.Voted.SetRootHash(nil)
	for _, c := range candidates {
		err := acc.Voted.Put(c.Bytes(), c.Bytes())
		if err != nil {
			return err
		}
	}
	return as.PutAccount(acc)
}

//SubVoted subtracts candidate from voted trie
func (as *AccountState) SubVoted(addr common.Address, candidate common.Address) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	_, err = acc.Voted.Get(candidate.Bytes())
	if err != nil || err == ErrNotFound {
		return err
	}
	if err != nil {
		return ErrNotVotedYet
	}

	err = acc.Voted.Delete(candidate.Bytes())
	if err != nil {
		return err
	}
	return as.PutAccount(acc)
}

//SetCollateral sets collateral
func (as *AccountState) SetCollateral(addr common.Address, collateral *util.Uint128) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Collateral = collateral
	return as.PutAccount(acc)
}

//AddVotePower adds vote power
func (as AccountState) AddVotePower(addr common.Address, amount *util.Uint128) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.VotePower, err = acc.VotePower.Add(amount)
	if err != nil {
		return err
	}
	return as.PutAccount(acc)
}

//SubVotePower subtracts vote power
func (as AccountState) SubVotePower(addr common.Address, amount *util.Uint128) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.VotePower, err = acc.VotePower.Sub(amount)
	if err != nil {
		return err
	}
	return as.PutAccount(acc)
}

//AddVoters adds voter's addr to candidate's voters trie
func (as AccountState) AddVoters(candidate common.Address, voter common.Address) error {
	acc, err := as.GetAccount(candidate)
	if err != nil {
		return err
	}
	_, err = acc.Voters.Get(voter.Bytes())
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrAlreadyInVoters
	}

	err = acc.Voters.Put(voter.Bytes(), voter.Bytes())
	if err != nil {
		return err
	}

	as.PutAccount(acc)
	acc, _ = as.GetAccount(candidate)

	return nil
}

//SubVoters subtracts voter's addr to candidate's voters trie
func (as AccountState) SubVoters(candidate common.Address, voter common.Address) error {
	acc, err := as.GetAccount(candidate)
	if err != nil {
		return err
	}
	_, err = acc.Voters.Get(voter.Bytes())
	if err == ErrNotFound {
		return ErrNotInVoters
	}
	if err != nil {
		return err
	}

	err = acc.Voters.Delete(voter.Bytes())
	if err != nil {
		return err
	}
	return as.PutAccount(acc)
}

// AddRecord adds a record hash in account's records list
func (as *AccountState) AddRecord(addr common.Address, recordHash []byte) error {
	var err error
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}

	_, err = acc.Records.Get(recordHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrRecordAlreadyAdded
	}

	acc.Records.Put(recordHash, recordHash)
	return as.PutAccount(acc)
}

// AddCertReceived adds a cert hash in certReceived
func (as *AccountState) AddCertReceived(addr common.Address, certHash []byte) error {

	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	_, err = acc.CertsReceived.Get(certHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrCertReceivedAlreadyAdded
	}

	acc.CertsReceived.Put(certHash, certHash)
	return as.PutAccount(acc)
}

// AddCertIssued adds a cert hash info in certIssued
func (as *AccountState) AddCertIssued(addr common.Address, certHash []byte) error {

	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	_, err = acc.CertsIssued.Get(certHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrCertIssuedAlreadyAdded
	}

	acc.CertsIssued.Put(certHash, certHash)
	return as.PutAccount(acc)
}

// AddTxsFrom add transaction in TxsFrom
func (as *AccountState) AddTxsFrom(addr common.Address, txHash []byte) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.TxsFrom.Put(txHash, txHash)
	return as.PutAccount(acc)
}

// AddTxsTo add transaction in TxsFrom
func (as *AccountState) AddTxsTo(addr common.Address, txHash []byte) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.TxsTo.Put(txHash, txHash)
	return as.PutAccount(acc)
}

//AddTxs add transaction to account state
func (as *AccountState) AddTxs(tx *Transaction) error {
	if err := as.AddTxsFrom(tx.From(), tx.Hash()); err != nil {
		return err
	}

	if err := as.AddTxsTo(tx.To(), tx.Hash()); err != nil {
		return err
	}

	return nil
}

//Accounts returns account slice
func (as *AccountState) Accounts() ([]*Account, error) {
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

		pbAccount := new(corepb.AccountState)
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
func KeyTrieToSlice(trie *trie.Trie) [][]byte {
	var slice [][]byte
	if trie.RootHash() == nil {
		return slice
	}
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
