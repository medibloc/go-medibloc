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

func (s *states) Clone() (*states, error) {
	accState, err := NewAccountStateBatch(s.accState.RootHash(), s.storage)
	if err != nil {
		return nil, err
	}

	txsState, err := trie.NewBatch(s.txsState.RootHash(), s.storage)
	if err != nil {
		return nil, err
	}

	usageState, err := trie.NewBatch(s.usageState.RootHash(), s.storage)
	if err != nil {
		return nil, err
	}

	recordsState, err := trie.NewBatch(s.recordsState.RootHash(), s.storage)
	if err != nil {
		return nil, err
	}

	dposState, err := DposState.Clone(s.dposState)
	if err != nil {
		return nil, err
	}

	certificationState, err := trie.NewBatch(s.certificationState.RootHash(), s.storage)
	if err != nil {
		return nil, err
	}

	reservationQueue, err := LoadReservationQueue(s.storage, s.reservationQueue.Hash())
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
		storage:            s.storage,
	}, nil
}

func (s *states) BeginBatch() error {
	if err := s.accState.BeginBatch(); err != nil {
		return err
	}
	if err := s.txsState.BeginBatch(); err != nil {
		return err
	}
	if err := s.usageState.BeginBatch(); err != nil {
		return err
	}
	if err := s.recordsState.BeginBatch(); err != nil {
		return err
	}
	if err := s.DposState().BeginBatch(); err != nil {
		return err
	}
	if err := s.certificationState.BeginBatch(); err != nil {
		return err
	}
	return s.reservationQueue.BeginBatch()
}

func (s *states) Commit() error {
	if err := s.accState.Commit(); err != nil {
		return err
	}
	if err := s.txsState.Commit(); err != nil {
		return err
	}
	if err := s.usageState.Commit(); err != nil {
		return err
	}
	if err := s.recordsState.Commit(); err != nil {
		return err
	}
	if err := s.dposState.Commit(); err != nil {
		return err
	}
	if err := s.certificationState.Commit(); err != nil {
		return err
	}
	return s.reservationQueue.Commit()
}

func (s *states) AccountsRoot() []byte {
	return s.accState.RootHash()
}

func (s *states) TransactionsRoot() []byte {
	return s.txsState.RootHash()
}

func (s *states) UsageRoot() []byte {
	return s.usageState.RootHash()
}

func (s *states) RecordsRoot() []byte {
	return s.recordsState.RootHash()
}

func (s *states) CertificationRoot() []byte {
	return s.certificationState.RootHash()
}

func (s *states) ReservationQueueHash() []byte {
	return s.reservationQueue.Hash()
}

func (s *states) LoadAccountsRoot(rootHash []byte) error {
	accState, err := NewAccountStateBatch(rootHash, s.storage)
	if err != nil {
		return err
	}
	s.accState = accState
	return nil
}

func (s *states) LoadTransactionsRoot(rootHash []byte) error {
	txsState, err := trie.NewBatch(rootHash, s.storage)
	if err != nil {
		return err
	}
	s.txsState = txsState
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

func (s *states) LoadRecordsRoot(rootHash []byte) error {
	recordsState, err := trie.NewBatch(rootHash, s.storage)
	if err != nil {
		return err
	}
	s.recordsState = recordsState
	return nil
}

func (s *states) LoadCertificationRoot(rootHash []byte) error {
	certificationState, err := trie.NewBatch(rootHash, s.storage)
	if err != nil {
		return err
	}
	s.certificationState = certificationState
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

func (s *states) GetAccount(address common.Address) (*Account, error) {
	return s.accState.GetAccount(address.Bytes())
}

func (s *states) GetAccounts() ([]*Account, error) {
	return s.accState.AccountState().Accounts()
}

func (s *states) AddBalance(address common.Address, amount *util.Uint128) error {
	return s.accState.AddBalance(address.Bytes(), amount)
}

func (s *states) SubBalance(address common.Address, amount *util.Uint128) error {
	return s.accState.SubBalance(address.Bytes(), amount)
}

func (s *states) AddTransaction(tx *Transaction) error {
	var err error

	err = s.accState.AddTransaction(tx.From().Bytes(), tx.Hash(), true)
	if err != nil && err != ErrTransactionHashAlreadyAdded {
		return err
	}

	err = s.accState.AddTransaction(tx.To().Bytes(), tx.Hash(), false)
	if err != nil && err != ErrTransactionHashAlreadyAdded {
		return err
	}

	return nil
}

func (s *states) AddRecord(tx *Transaction, hash []byte, owner common.Address) error {
	record := &corepb.Record{
		Hash:      hash,
		Owner:     tx.from.Bytes(),
		Timestamp: tx.Timestamp(),
	}
	recordBytes, err := proto.Marshal(record)
	if err != nil {
		return err
	}

	if err := s.recordsState.Put(hash, recordBytes); err != nil {
		return err
	}

	return s.accState.AddRecord(tx.from.Bytes(), hash)
}

func (s *states) GetRecord(hash []byte) (*corepb.Record, error) {
	recordBytes, err := s.recordsState.Get(hash)
	if err != nil {
		return nil, err
	}
	pbRecord := new(corepb.Record)
	if err := proto.Unmarshal(recordBytes, pbRecord); err != nil {
		return nil, err
	}
	return pbRecord, nil
}

func (s *states) incrementNonce(address common.Address) error {
	return s.accState.IncrementNonce(address.Bytes())
}

func (s *states) GetTx(txHash []byte) ([]byte, error) {
	return s.txsState.Get(txHash)
}

func (s *states) PutTx(txHash []byte, txBytes []byte) error {
	return s.txsState.Put(txHash, txBytes)
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
		return s.usageState.Put(payer.Bytes(), usageBytes)
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

//Certification returns certification for hash
func (s *states) Certification(hash []byte) (*corepb.Certification, error) {
	certBytes, err := s.certificationState.Get(hash)
	if err != nil {
		return nil, err
	}
	pbCert := new(corepb.Certification)
	err = proto.Unmarshal(certBytes, pbCert)
	if err != nil {
		return nil, err
	}
	return pbCert, nil
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
	fromAcc, err := bs.AccState().GetAccount(tx.from.Bytes())
	if err != nil {
		return err
	}

	expectedNonce := fromAcc.nonce + 1
	if tx.nonce > expectedNonce {
		return ErrLargeTransactionNonce
	} else if tx.nonce < expectedNonce {
		return ErrSmallTransactionNonce
	}
	return nil
}
