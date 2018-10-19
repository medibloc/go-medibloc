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
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// BlockState is block state
type BlockState struct {
	reward *util.Uint128
	supply *util.Uint128

	accState  *AccountState
	txState   *TransactionState
	dposState DposState

	storage storage.Storage
}

// Supply returns supply in state
func (bs *BlockState) Supply() *util.Uint128 {
	return bs.supply
}

// Reward returns reward in state
func (bs *BlockState) Reward() *util.Uint128 {
	return bs.reward
}

// AccState returns account state in state
func (bs *BlockState) AccState() *AccountState {
	return bs.accState
}

// DposState returns dpos state in state
func (bs *BlockState) DposState() DposState {
	return bs.dposState
}

// GetCandidates returns list of candidates (only used in grpc)
func (bs *BlockState) GetCandidates() ([]common.Address, error) {
	// TODO: should be deprecate (if candidates are too many, grpc cannot full list of cadidiates. (max msg size)
	return bs.DposState().Candidates()
}

// GetDynasty returns list of dynasty (only used in grpc)
func (bs *BlockState) GetDynasty() ([]common.Address, error) { // TODO: deprecate ?

	return bs.DposState().Dynasty()
}

func newStates(consensus Consensus, stor storage.Storage) (*BlockState, error) {
	accState, err := NewAccountState(nil, stor)
	if err != nil {
		return nil, err
	}

	txState, err := NewTransactionState(nil, stor)
	if err != nil {
		return nil, err
	}

	dposState, err := consensus.NewConsensusState(nil, stor)
	if err != nil {
		return nil, err
	}

	return &BlockState{
		reward:    util.NewUint128(),
		supply:    util.NewUint128(),
		accState:  accState,
		txState:   txState,
		dposState: dposState,
		storage:   stor,
	}, nil
}

//Clone clone states
func (bs *BlockState) Clone() (*BlockState, error) {
	accState, err := bs.accState.Clone()
	if err != nil {
		return nil, err
	}

	txState, err := bs.txState.Clone()
	if err != nil {
		return nil, err
	}

	dposState, err := bs.dposState.Clone()
	if err != nil {
		return nil, err
	}

	return &BlockState{
		reward:    bs.reward.DeepCopy(),
		supply:    bs.supply.DeepCopy(),
		accState:  accState,
		txState:   txState,
		dposState: dposState,
		storage:   bs.storage,
	}, nil
}

func (bs *BlockState) prepare() error {
	if err := bs.accState.Prepare(); err != nil {
		return err
	}
	if err := bs.txState.Prepare(); err != nil {
		return err
	}
	if err := bs.DposState().Prepare(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) beginBatch() error {
	if err := bs.accState.BeginBatch(); err != nil {
		return err
	}
	if err := bs.txState.BeginBatch(); err != nil {
		return err
	}
	if err := bs.DposState().BeginBatch(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) commit() error {
	if err := bs.accState.Commit(); err != nil {
		return err
	}
	if err := bs.txState.Commit(); err != nil {
		return err
	}
	if err := bs.dposState.Commit(); err != nil {
		return err
	}
	return nil
}
func (bs *BlockState) rollBack() error {
	if err := bs.accState.RollBack(); err != nil {
		return err
	}
	if err := bs.txState.RollBack(); err != nil {
		return err
	}
	if err := bs.dposState.RollBack(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) flush() error {
	if err := bs.accState.Flush(); err != nil {
		return err
	}
	if err := bs.txState.Flush(); err != nil {
		return err
	}
	if err := bs.dposState.Flush(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) reset() error {
	if err := bs.accState.Reset(); err != nil {
		return err
	}
	if err := bs.txState.Reset(); err != nil {
		return err
	}
	if err := bs.dposState.Reset(); err != nil {
		return err
	}
	return nil
}

//AccountsRoot returns account state root
func (bs *BlockState) AccountsRoot() ([]byte, error) {
	return bs.accState.RootHash()
}

//TxsRoot returns transaction state root
func (bs *BlockState) TxsRoot() ([]byte, error) {
	return bs.txState.RootHash()
}

//DposRoot returns dpos state root
func (bs *BlockState) DposRoot() ([]byte, error) {
	return bs.dposState.RootBytes()
}

func (bs *BlockState) loadAccountState(rootHash []byte) error {
	accState, err := NewAccountState(rootHash, bs.storage)
	if err != nil {
		return err
	}
	bs.accState = accState
	return nil
}

func (bs *BlockState) loadTransactionState(rootBytes []byte) error {
	txState, err := NewTransactionState(rootBytes, bs.storage)
	if err != nil {
		return err
	}
	bs.txState = txState
	return nil
}

//GetAccount returns account in state
func (bs *BlockState) GetAccount(addr common.Address) (*Account, error) {
	return bs.accState.GetAccount(addr)
}

//PutAccount put account to state
func (bs *BlockState) PutAccount(acc *Account) error {
	return bs.accState.putAccount(acc)
}

//GetTx returns txs in state
func (bs *BlockState) GetTx(txHash []byte) (*Transaction, error) {
	return bs.txState.Get(txHash)
}

func (bs *BlockState) incrementNonce(address common.Address) error {
	return bs.accState.incrementNonce(address)
}

func (bs *BlockState) acceptTransaction(tx *Transaction) error {
	if err := bs.txState.Put(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to put a transaction to transaction state.")
		return err
	}

	if err := bs.accState.PutTx(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to put a transaction to account")
		return err
	}

	return bs.incrementNonce(tx.from)
}

// checkNonce compare given transaction's nonce with expected account's nonce
func (bs *BlockState) checkNonce(tx *Transaction) error {
	fromAcc, err := bs.GetAccount(tx.from)
	if err != nil {
		return err
	}

	expectedNonce := fromAcc.Nonce + 1
	if tx.nonce > expectedNonce {
		return ErrLargeTransactionNonce
	} else if tx.nonce < expectedNonce {
		return ErrSmallTransactionNonce
	}
	return nil
}
