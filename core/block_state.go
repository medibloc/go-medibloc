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
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/medibloc/go-medibloc/common/trie"
)

type states struct {
	accState           *AccountStateBatch
	txsState           *trie.Batch
	usageState         *trie.Batch
	recordsState       *trie.Batch
	dposState          DposState
	certificationState *trie.Batch

	reservationQueue *ReservationQueue

	storage storage.Storage
}

func (s *states) AccState() *AccountStateBatch {
	return s.accState
}

func (s *states) DposState() DposState {
	return s.dposState
}

func newStates(consensus Consensus, stor storage.Storage) (*states, error) {
	accState, err := NewAccountStateBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	txsState, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	usageState, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	recordsState, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	dposState, err := consensus.NewConsensusState(nil, stor)
	if err != nil {
		return nil, err
	}

	certificationState, err := trie.NewBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	reservationQueue := NewEmptyReservationQueue(stor)

	return &states{
		accState:           accState,
		txsState:           txsState,
		usageState:         usageState,
		recordsState:       recordsState,
		dposState:          dposState,
		certificationState: certificationState,
		reservationQueue:   reservationQueue,
		storage:            stor,
	}, nil
}

func (st *states) Clone() (*states, error) {
	accState, err := NewAccountStateBatch(st.accState.RootHash(), st.storage)
	if err != nil {
		return nil, err
	}

	txsState, err := trie.NewBatch(st.txsState.RootHash(), st.storage)
	if err != nil {
		return nil, err
	}

	usageState, err := trie.NewBatch(st.usageState.RootHash(), st.storage)
	if err != nil {
		return nil, err
	}

	recordsState, err := trie.NewBatch(st.recordsState.RootHash(), st.storage)
	if err != nil {
		return nil, err
	}

	dposState, err := DposState.Clone(st.dposState)
	if err != nil {
		return nil, err
	}

	certificationState, err := trie.NewBatch(st.certificationState.RootHash(), st.storage)
	if err != nil {
		return nil, err
	}

	reservationQueue, err := LoadReservationQueue(st.storage, st.reservationQueue.Hash())
	if err != nil {
		return nil, err
	}

	return &states{
		accState:           accState,
		txsState:           txsState,
		usageState:         usageState,
		recordsState:       recordsState,
		dposState:          dposState,
		certificationState: certificationState,
		reservationQueue:   reservationQueue,
		storage:            st.storage,
	}, nil
}

func (st *states) BeginBatch() error {
	if err := st.accState.BeginBatch(); err != nil {
		return err
	}
	if err := st.txsState.BeginBatch(); err != nil {
		return err
	}
	if err := st.usageState.BeginBatch(); err != nil {
		return err
	}
	if err := st.recordsState.BeginBatch(); err != nil {
		return err
	}
	if err := st.DposState().BeginBatch(); err != nil {
		return err
	}
	if err := st.certificationState.BeginBatch(); err != nil {
		return err
	}
	return st.reservationQueue.BeginBatch()
}

func (st *states) Commit() error {
	if err := st.accState.Commit(); err != nil {
		return err
	}
	if err := st.txsState.Commit(); err != nil {
		return err
	}
	if err := st.usageState.Commit(); err != nil {
		return err
	}
	if err := st.recordsState.Commit(); err != nil {
		return err
	}
	if err := st.dposState.Commit(); err != nil {
		return err
	}
	if err := st.certificationState.Commit(); err != nil {
		return err
	}
	return st.reservationQueue.Commit()
}

func (st *states) AccountsRoot() []byte {
	return st.accState.RootHash()
}

func (st *states) TransactionsRoot() []byte {
	return st.txsState.RootHash()
}

func (st *states) UsageRoot() []byte {
	return st.usageState.RootHash()
}

func (st *states) RecordsRoot() []byte {
	return st.recordsState.RootHash()
}

func (st *states) CertificationRoot() []byte {
	return st.certificationState.RootHash()
}

func (st *states) ReservationQueueHash() []byte {
	return st.reservationQueue.Hash()
}

func (st *states) LoadAccountsRoot(rootHash []byte) error {
	accState, err := NewAccountStateBatch(rootHash, st.storage)
	if err != nil {
		return err
	}
	st.accState = accState
	return nil
}

func (st *states) LoadTransactionsRoot(rootHash []byte) error {
	txsState, err := trie.NewBatch(rootHash, st.storage)
	if err != nil {
		return err
	}
	st.txsState = txsState
	return nil
}

func (st *states) LoadUsageRoot(rootHash []byte) error {
	usageState, err := trie.NewBatch(rootHash, st.storage)
	if err != nil {
		return err
	}
	st.usageState = usageState
	return nil
}

func (st *states) LoadRecordsRoot(rootHash []byte) error {
	recordsState, err := trie.NewBatch(rootHash, st.storage)
	if err != nil {
		return err
	}
	st.recordsState = recordsState
	return nil
}


func (st *states) LoadCertificationRoot(rootHash []byte) error {
	certificationState, err := trie.NewBatch(rootHash, st.storage)
	if err != nil {
		return err
	}
	st.certificationState = certificationState
	return nil
}

func (st *states) LoadReservationQueue(hash []byte) error {
	rq, err := LoadReservationQueue(st.storage, hash)
	if err != nil {
		return err
	}
	st.reservationQueue = rq
	return nil
}


func (st *states) GetAccount(address common.Address) (Account, error) {
	return st.accState.GetAccount(address.Bytes())
}

func (st *states) AddBalance(address common.Address, amount *util.Uint128) error {
	return st.accState.AddBalance(address.Bytes(), amount)
}

func (st *states) SubBalance(address common.Address, amount *util.Uint128) error {
	return st.accState.SubBalance(address.Bytes(), amount)
}

func (st *states) AddRecord(tx *Transaction, hash []byte, owner common.Address) error {
	record := &corepb.Record{
		Hash:      hash,
		Owner:     tx.from.Bytes(),
		Timestamp: tx.Timestamp(),
	}
	recordBytes, err := proto.Marshal(record)
	if err != nil {
		return err
	}

	if err := st.recordsState.Put(hash, recordBytes); err != nil {
		return err
	}

	return st.accState.AddRecord(tx.from.Bytes(), hash)
}

func (st *states) GetRecord(hash []byte) (*corepb.Record, error) {
	recordBytes, err := st.recordsState.Get(hash)
	if err != nil {
		return nil, err
	}
	pbRecord := new(corepb.Record)
	if err := proto.Unmarshal(recordBytes, pbRecord); err != nil {
		return nil, err
	}
	return pbRecord, nil
}

func (st *states) incrementNonce(address common.Address) error {
	return st.accState.IncrementNonce(address.Bytes())
}

func (st *states) GetTx(txHash []byte) ([]byte, error) {
	return st.txsState.Get(txHash)
}

func (st *states) PutTx(txHash []byte, txBytes []byte) error {
	return st.txsState.Put(txHash, txBytes)
}

func (st *states) updateUsage(tx *Transaction, blockTime int64) error {
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

	usageBytes, err := st.usageState.Get(payer.Bytes())
	switch err {
	case nil:
	case ErrNotFound:
		usage := &corepb.Usage{
			Timestamps: []*corepb.TxTimestamp{
				{
					Hash:      tx.Hash(),
					Timestamp: tx.Timestamp(),
				},
			},
		}
		usageBytes, err = proto.Marshal(usage)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"usage": usage,
				"err":   err,
			}).Error("Failed to marshal usage.")
			return err
		}
		return st.usageState.Put(payer.Bytes(), usageBytes)
	default:
		logging.Console().WithFields(logrus.Fields{
			"payer": payer.Hex(),
			"err":   err,
		}).Error("Failed to get usage from trie.")
		return err
	}

	pbUsage := new(corepb.Usage)
	if err := proto.Unmarshal(usageBytes, pbUsage); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"pb":  pbUsage,
		}).Error("Failed to unmarshal proto.")
		return err
	}

	var idx int
	for idx = range pbUsage.Timestamps {
		if blockTime-weekSec < tx.Timestamp() {
			break
		}
	}
	pbUsage.Timestamps = append(pbUsage.Timestamps[idx:], &corepb.TxTimestamp{Hash: tx.Hash(), Timestamp: tx.Timestamp()})
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

	return st.usageState.Put(payer.Bytes(), pbBytes)
}

func (st *states) GetUsage(addr common.Address) ([]*corepb.TxTimestamp, error) {
	usageBytes, err := st.usageState.Get(addr.Bytes())
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
func (st *states) GetReservedTasks() []*ReservedTask {
	return st.reservationQueue.Tasks()
}

// AddReservedTask adds a reserved task in reservation queue
func (st *states) AddReservedTask(task *ReservedTask) error {
	return st.reservationQueue.AddTask(task)
}

// PopReservedTask pops reserved tasks which should be processed before 'before'
func (st *states) PopReservedTasks(before int64) []*ReservedTask {
	return st.reservationQueue.PopTasksBefore(before)
}

func (st *states) PeekHeadReservedTask() *ReservedTask {
	return st.reservationQueue.Peek()
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
	bs.snapshot = nil
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

	if err := bs.PutTx(tx.hash, txBytes); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
			"tx":  tx,
		}).Error("Failed to put a transaction to block state.")
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
	fromAcc, err := bs.GetAccount(tx.from)
	if err != nil {
		return err
	}

	expectedNonce := fromAcc.Nonce() + 1
	if tx.nonce > expectedNonce {
		return ErrLargeTransactionNonce
	} else if tx.nonce < expectedNonce {
		return ErrSmallTransactionNonce
	}
	return nil
}
