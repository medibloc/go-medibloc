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
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type states struct {
	reward *util.Uint128
	supply *util.Uint128

	accState         *AccountState
	dataState        *DataState
	dposState        DposState
	usageState       *trie.Batch
	reservationQueue *ReservationQueue

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

	dataState, err := NewDataState(nil, stor)
	if err != nil {
		return nil, err
	}

	dposState, err := consensus.NewConsensusState(nil, stor)
	if err != nil {
		return nil, err
	}

	usageState, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	reservationQueue := NewEmptyReservationQueue(stor)

	return &states{
		reward:           util.NewUint128(),
		supply:           util.NewUint128(),
		accState:         accState,
		dataState:        dataState,
		dposState:        dposState,
		usageState:       usageState,
		reservationQueue: reservationQueue,
		storage:          stor,
	}, nil
}

func (s *states) Clone() (*states, error) {
	accState, err := NewAccountState(s.accState.RootHash(), s.storage)
	if err != nil {
		return nil, err
	}

	dsBytes, err := s.dataState.RootBytes()
	if err != nil {
		return nil, err
	}
	dataState, err := NewDataState(dsBytes, s.storage)
	if err != nil {
		return nil, err
	}

	dposState, err := DposState.Clone(s.dposState)
	if err != nil {
		return nil, err
	}

	usageState, err := trie.NewBatch(s.usageState.RootHash(), s.storage)
	if err != nil {
		return nil, err
	}

	reservationQueue, err := LoadReservationQueue(s.storage, s.reservationQueue.Hash())
	if err != nil {
		return nil, err
	}

	return &states{
		reward:           s.reward.DeepCopy(),
		supply:           s.supply.DeepCopy(),
		accState:         accState,
		dataState:        dataState,
		dposState:        dposState,
		usageState:       usageState,
		reservationQueue: reservationQueue,
		storage:          s.storage,
	}, nil
}

func (s *states) BeginBatch() error {
	if err := s.accState.BeginBatch(); err != nil {
		return err
	}
	if err := s.dataState.BeginBatch(); err != nil {
		return err
	}
	if err := s.DposState().BeginBatch(); err != nil {
		return err
	}
	if err := s.usageState.BeginBatch(); err != nil {
		return err
	}
	return s.reservationQueue.BeginBatch()
}

func (s *states) Commit() error {
	if err := s.accState.Commit(); err != nil {
		return err
	}
	if err := s.dataState.Commit(); err != nil {
		return err
	}
	if err := s.dposState.Commit(); err != nil {
		return err
	}
	if err := s.usageState.Commit(); err != nil {
		return err
	}
	return s.reservationQueue.Commit()
}

func (s *states) AccountsRoot() []byte {
	return s.accState.RootHash()
}

func (s *states) DataRoot() ([]byte, error) {
	return s.dataState.RootBytes()
}

func (s *states) DposRoot() ([]byte, error) {
	return s.dposState.RootBytes()
}

func (s *states) UsageRoot() []byte {
	return s.usageState.RootHash()
}

func (s *states) ReservationQueueHash() []byte {
	return s.reservationQueue.Hash()
}

func (s *states) LoadAccountState(rootHash []byte) error {
	accState, err := NewAccountState(rootHash, s.storage)
	if err != nil {
		return err
	}
	s.accState = accState
	return nil
}

func (s *states) LoadDataState(rootBytes []byte) error {
	dataState, err := NewDataState(rootBytes, s.storage)
	if err != nil {
		return err
	}
	s.dataState = dataState
	return nil
}

func (s *states) LoadUsageRoot(rootHash []byte) error {
	usageState, err := trie.NewBatch(rootHash, s.storage)
	if err != nil {
		return err
	}
	s.usageState = usageState
	return nil
}

func (s *states) LoadReservationQueue(hash []byte) error {
	rq, err := LoadReservationQueue(s.storage, hash)
	if err != nil {
		return err
	}
	s.reservationQueue = rq
	return nil
}

func (s *states) DataState() *DataState {
	return s.dataState
}

func (s *states) GetAccount(addr common.Address) (*Account, error) {
	return s.accState.GetAccount(addr)
}

func (s *states) GetAccounts() ([]*Account, error) {
	return s.accState.Accounts()
}

func (s *states) AddBalance(address common.Address, amount *util.Uint128) error {
	return s.accState.AddBalance(address, amount)
}

func (s *states) SubBalance(address common.Address, amount *util.Uint128) error {
	return s.accState.SubBalance(address, amount)
}

func (s *states) incrementNonce(address common.Address) error {
	return s.accState.IncrementNonce(address)
}

func (s *states) updateUsage(tx *Transaction, blockTime int64) error {
	weekSec := int64(604800)

	if tx.Timestamp() < blockTime-weekSec {
		return ErrTooOldTransaction
	}

	payer, err := tx.recoverPayer()
	if err == ErrPayerSignatureNotExist {
		payer = tx.from
	} else if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to recover payer address.")
		return err
	}

	usageBytes, err := s.usageState.Get(payer.Bytes())
	if err != nil && err != ErrNotFound {
		logging.Console().WithFields(logrus.Fields{
			"payer": payer.Hex(),
			"err":   err,
		}).Error("Failed to get usage from trie.")
		return err
	}

	var pbUsage *corepb.Usage
	if err != nil && err == ErrNotFound {
		pbUsage = &corepb.Usage{
			Timestamps: []*corepb.TxTimestamp{},
		}
	} else {
		pbUsage = new(corepb.Usage)
		if err := proto.Unmarshal(usageBytes, pbUsage); err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
				"pb":  pbUsage,
			}).Error("Failed to unmarshal proto.")
			return err
		}
	}

	for i, ttx := range pbUsage.Timestamps {
		if blockTime-weekSec > ttx.Timestamp {
			pbUsage.Timestamps = pbUsage.Timestamps[i+1:]
			break
		}
	}

	pbUsage.Timestamps = append(pbUsage.Timestamps, &corepb.TxTimestamp{
		Hash:      tx.Hash(),
		Timestamp: tx.Timestamp(),
	})

	sort.Slice(pbUsage.Timestamps, func(i, j int) bool {
		return pbUsage.Timestamps[i].Timestamp < pbUsage.Timestamps[j].Timestamp
	})

	pbBytes, err := proto.Marshal(pbUsage)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"pb":  pbUsage,
		}).Error("Failed to marshal proto.")
		return err
	}

	return s.usageState.Put(payer.Bytes(), pbBytes)
}

func (s *states) GetUsage(addr common.Address) ([]*corepb.TxTimestamp, error) {
	usageBytes, err := s.usageState.Get(addr.Bytes())
	switch err {
	case nil:
	case ErrNotFound:
		return []*corepb.TxTimestamp{}, nil
	default:
		return nil, err
	}

	pbUsage := new(corepb.Usage)
	if err := proto.Unmarshal(usageBytes, pbUsage); err != nil {
		return nil, err
	}
	return pbUsage.Timestamps, nil
}

// GetReservedTasks returns reserved tasks in reservation queue
func (s *states) GetReservedTasks() []*ReservedTask {
	return s.reservationQueue.Tasks()
}

// AddReservedTask adds a reserved task in reservation queue
func (s *states) AddReservedTask(task *ReservedTask) error {
	return s.reservationQueue.AddTask(task)
}

// PopReservedTasks pops reserved tasks which should be processed before 'before'
func (s *states) PopReservedTasks(before int64) []*ReservedTask {
	return s.reservationQueue.PopTasksBefore(before)
}

func (s *states) PeekHeadReservedTask() *ReservedTask {
	return s.reservationQueue.Peek()
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

// AcceptTransaction and update internal txsStates
func (bs *BlockState) AcceptTransaction(tx *Transaction, blockTime int64) error {
	pbTx, err := tx.ToProto()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to convert a transaction to proto.")
		return err
	}

	txBytes, err := proto.Marshal(pbTx)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"pb":  pbTx,
		}).Error("Failed to marshal proto.")
		return err
	}

	if err := bs.dataState.PutTx(tx.hash, txBytes); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to put a transaction to transaction state.")
		return err
	}

	if err := bs.accState.AddTxs(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to put a transaction to account")
		return err
	}

	if err := bs.updateUsage(tx, blockTime); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"tx":        tx,
			"blockTime": blockTime,
		}).Error("Failed to update usage.")
		return err
	}

	return bs.incrementNonce(tx.from)
}

func (bs *BlockState) checkNonce(tx *Transaction) error {
	fromAcc, err := bs.AccState().GetAccount(tx.from)
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
