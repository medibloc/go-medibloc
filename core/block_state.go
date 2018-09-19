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

type states struct {
	reward *util.Uint128
	supply *util.Uint128

	accState  *AccountState
	txState   *TransactionState
	dposState DposState

	storage storage.Storage
}

func (s *states) Supply() *util.Uint128 {
	return s.supply.DeepCopy()
}

func (s *states) Reward() *util.Uint128 {
	return s.reward.DeepCopy()
}

func (s *states) AccState() *AccountState {
	return s.accState
}

func (s *states) DposState() DposState {
	return s.dposState
}

func (s *states) GetCandidates() ([]common.Address, error) {
	return s.DposState().Candidates()
}

func (s *states) GetDynasty() ([]common.Address, error) {
	return s.DposState().Dynasty()
}

func newStates(consensus Consensus, stor storage.Storage) (*states, error) {
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

	return &states{
		reward:    util.NewUint128(),
		supply:    util.NewUint128(),
		accState:  accState,
		txState:   txState,
		dposState: dposState,
		storage:   stor,
	}, nil
}

func (s *states) Clone() (*states, error) {
	accState, err := s.accState.Clone()
	if err != nil {
		return nil, err
	}

	txState, err := s.txState.Clone()
	if err != nil {
		return nil, err
	}

	dposState, err := s.dposState.Clone()
	if err != nil {
		return nil, err
	}

	return &states{
		reward:    s.reward.DeepCopy(),
		supply:    s.supply.DeepCopy(),
		accState:  accState,
		txState:   txState,
		dposState: dposState,
		storage:   s.storage,
	}, nil
}

func (s *states) BeginBatch() error {
	if err := s.accState.BeginBatch(); err != nil {
		return err
	}
	if err := s.txState.BeginBatch(); err != nil {
		return err
	}
	if err := s.DposState().BeginBatch(); err != nil {
		return err
	}
	return nil
}

func (s *states) Commit() error {
	if err := s.accState.Commit(); err != nil {
		return err
	}
	if err := s.txState.Commit(); err != nil {
		return err
	}
	if err := s.dposState.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *states) AccountsRoot() ([]byte, error) {
	return s.accState.RootHash()
}

func (s *states) TxsRoot() ([]byte, error) {
	return s.txState.RootHash()
}

func (s *states) DposRoot() ([]byte, error) {
	return s.dposState.RootBytes()
}

func (s *states) LoadAccountState(rootHash []byte) error {
	accState, err := NewAccountState(rootHash, s.storage)
	if err != nil {
		return err
	}
	s.accState = accState
	return nil
}

func (s *states) LoadTransactionState(rootBytes []byte) error {
	txState, err := NewTransactionState(rootBytes, s.storage)
	if err != nil {
		return err
	}
	s.txState = txState
	return nil
}

func (s *states) GetAccount(addr common.Address) (*Account, error) {
	return s.accState.GetAccount(addr)
}

func (s *states) PutAccount(acc *Account) error {
	return s.accState.putAccount(acc)
}

func (s *states) GetAccounts() ([]*Account, error) {
	return s.accState.Accounts()
}

func (s *states) GetTx(txHash []byte) (*Transaction, error) {
	return s.txState.Get(txHash)
}

func (s *states) incrementNonce(address common.Address) error {
	return s.accState.incrementNonce(address)
}

func (s *states) acceptTransaction(tx *Transaction, blockTime int64) error {
	if err := s.txState.Put(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to put a transaction to transaction state.")
		return err
	}

	if err := s.accState.PutTx(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to put a transaction to account")
		return err
	}

	return s.incrementNonce(tx.from)
}

func (s *states) checkNonce(tx *Transaction) error {
	fromAcc, err := s.GetAccount(tx.from)
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

// BlockState possesses every states a block should have
type BlockState struct {
	*states
	snapshot *states
}

// NewBlockState creates a new block state
func NewBlockState(consensus Consensus, stor storage.Storage) (*BlockState, error) {
	states, err := newStates(consensus, stor)
	if err != nil {
		return nil, err
	}
	return &BlockState{
		states:   states,
		snapshot: nil,
	}, nil
}

// Clone clones block state
func (bs *BlockState) Clone() (*BlockState, error) {
	states, err := bs.states.Clone()
	if err != nil {
		return nil, err
	}
	return &BlockState{
		states:   states,
		snapshot: nil,
	}, nil
}

// BeginBatch begins batch
func (bs *BlockState) BeginBatch() error {
	snapshot, err := bs.states.Clone()
	if err != nil {
		return err
	}
	if err := bs.states.BeginBatch(); err != nil {
		return err
	}
	bs.snapshot = snapshot
	return nil
}

// RollBack rolls back batch
func (bs *BlockState) RollBack() error {
	bs.states = bs.snapshot
	//bs.snapshot = nil
	return nil
}

// Commit saves batch updates
func (bs *BlockState) Commit() error {
	if err := bs.states.Commit(); err != nil {
		return err
	}
	bs.snapshot = nil
	return nil
}
