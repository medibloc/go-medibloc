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
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

// BlockHeader is block header
type BlockHeader struct {
	hash       []byte
	parentHash []byte

	accsRoot          []byte
	txsRoot           []byte
	usageRoot         []byte
	recordsRoot       []byte
	certificationRoot []byte
	dposRoot          []byte

	reservationQueueHash []byte

	coinbase  common.Address
	timestamp int64
	chainID   uint32

	alg  algorithm.Algorithm
	sign []byte
}

// ToProto converts BlockHeader to corepb.BlockHeader
func (b *BlockHeader) ToProto() (proto.Message, error) {
	return &corepb.BlockHeader{
		Hash:                 b.hash,
		ParentHash:           b.parentHash,
		Coinbase:             b.coinbase.Bytes(),
		Timestamp:            b.timestamp,
		ChainId:              b.chainID,
		Alg:                  uint32(b.alg),
		Sign:                 b.sign,
		AccsRoot:             b.accsRoot,
		TxsRoot:              b.txsRoot,
		UsageRoot:            b.usageRoot,
		RecordsRoot:          b.recordsRoot,
		CertificationRoot:    nil,
		DposRoot:             b.dposRoot,
		ReservationQueueHash: b.reservationQueueHash,
	}, nil
}

// FromProto converts corepb.BlockHeader to BlockHeader
func (b *BlockHeader) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.BlockHeader); ok {
		b.hash = msg.Hash
		b.parentHash = msg.ParentHash
		b.accsRoot = msg.AccsRoot
		b.txsRoot = msg.TxsRoot
		b.usageRoot = msg.UsageRoot
		b.recordsRoot = msg.RecordsRoot
		b.certificationRoot = msg.CertificationRoot
		b.dposRoot = msg.DposRoot
		b.reservationQueueHash = msg.ReservationQueueHash
		b.coinbase = common.BytesToAddress(msg.Coinbase)
		b.timestamp = msg.Timestamp
		b.chainID = msg.ChainId
		b.alg = algorithm.Algorithm(msg.Alg)
		b.sign = msg.Sign
		return nil
	}
	return ErrInvalidProtoToBlockHeader
}

func (b *BlockHeader) Hash() []byte {
	return b.hash
}

func (b *BlockHeader) SetHash(hash []byte) {
	b.hash = hash
}

func (b *BlockHeader) ParentHash() []byte {
	return b.parentHash
}

func (b *BlockHeader) SetParentHash(parentHash []byte) {
	b.parentHash = parentHash
}

func (b *BlockHeader) AccsRoot() []byte {
	return b.accsRoot
}

func (b *BlockHeader) SetAccsRoot(accsRoot []byte) {
	b.accsRoot = accsRoot
}

func (b *BlockHeader) TxsRoot() []byte {
	return b.txsRoot
}

func (b *BlockHeader) SetTxsRoot(txsRoot []byte) {
	b.txsRoot = txsRoot
}

func (b *BlockHeader) UsageRoot() []byte {
	return b.usageRoot
}

func (b *BlockHeader) SetUsageRoot(usageRoot []byte) {
	b.usageRoot = usageRoot
}

func (b *BlockHeader) RecordsRoot() []byte {
	return b.recordsRoot
}

func (b *BlockHeader) SetRecordsRoot(recordsRoot []byte) {
	b.recordsRoot = recordsRoot
}

func (b *BlockHeader) CertificationRoot() []byte {
	return b.certificationRoot
}

func (b *BlockHeader) SetCertificationRoot(certificationRoot []byte) {
	b.certificationRoot = certificationRoot
}

func (b *BlockHeader) DposRoot() []byte {
	return b.dposRoot
}

func (b *BlockHeader) SetDposRoot(dposRoot []byte) {
	b.dposRoot = dposRoot
}

func (b *BlockHeader) ReservationQueueHash() []byte {
	return b.reservationQueueHash
}

func (b *BlockHeader) SetReservationQueueHash(reservationQueueHash []byte) {
	b.reservationQueueHash = reservationQueueHash
}

func (b *BlockHeader) Coinbase() common.Address {
	return b.coinbase
}

func (b *BlockHeader) SetCoinbase(coinbase common.Address) {
	b.coinbase = coinbase
}

func (b *BlockHeader) Timestamp() int64 {
	return b.timestamp
}

func (b *BlockHeader) SetTimestamp(timestamp int64) {
	b.timestamp = timestamp
}

func (b *BlockHeader) ChainID() uint32 {
	return b.chainID
}

func (b *BlockHeader) SetChainID(chainID uint32) {
	b.chainID = chainID
}

func (b *BlockHeader) Alg() algorithm.Algorithm {
	return b.alg
}

func (b *BlockHeader) SetAlg(alg algorithm.Algorithm) {
	b.alg = alg
}

func (b *BlockHeader) Sign() []byte {
	return b.sign
}

func (b *BlockHeader) SetSign(sign []byte) {
	b.sign = sign
}

// BlockData represents a block
type BlockData struct {
	*BlockHeader
	transactions Transactions
	height       uint64
}

// ToProto converts Block to corepb.Block
func (bd *BlockData) ToProto() (proto.Message, error) {
	header, err := bd.BlockHeader.ToProto()
	if err != nil {
		return nil, err
	}
	if header, ok := header.(*corepb.BlockHeader); ok {
		txs := make([]*corepb.Transaction, len(bd.transactions))
		for idx, v := range bd.transactions {
			tx, err := v.ToProto()
			if err != nil {
				return nil, err
			}
			if tx, ok := tx.(*corepb.Transaction); ok {
				txs[idx] = tx
			} else {
				return nil, ErrCannotConvertTransaction
			}
		}
		return &corepb.Block{
			Header:       header,
			Transactions: txs,
			Height:       bd.height,
		}, nil
	}
	return nil, ErrInvalidBlockToProto
}

// FromProto converts corepb.Block to Block
func (bd *BlockData) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Block); ok {
		bd.BlockHeader = new(BlockHeader)
		if err := bd.BlockHeader.FromProto(msg.Header); err != nil {
			return err
		}

		bd.transactions = make(Transactions, len(msg.Transactions))
		for idx, v := range msg.Transactions {
			tx := new(Transaction)
			if err := tx.FromProto(v); err != nil {
				return err
			}
			bd.transactions[idx] = tx
		}
		bd.height = msg.Height
		return nil
	}
	return ErrInvalidProtoToBlock
}

func (bd *BlockData) Clone() (*BlockData, error) {
	protoBd, err := bd.ToProto()
	if err != nil {
		return nil, err
	}

	newBd := new(BlockData)
	err = newBd.FromProto(protoBd)
	if err != nil {
		return nil, err
	}
	return newBd, nil
}

// Height returns height
func (bd *BlockData) Height() uint64 {
	return bd.height
}

// SetHeight sets height.
func (bd *BlockData) SetHeight(height uint64) {
	bd.height = height
}

// Transactions returns txs in block
func (bd *BlockData) Transactions() Transactions {
	return bd.transactions
}

// SetTransactions sets transactions TO BE REMOVED: For test without block pool
func (bd *BlockData) SetTransactions(txs Transactions) error {
	bd.transactions = txs
	return nil
}

// String implements Stringer interface.
func (bd *BlockData) String() string {
	return fmt.Sprintf("<Height:%v, Hash:%v, ParentHash:%v>", bd.Height(), byteutils.Bytes2Hex(bd.Hash()), byteutils.Bytes2Hex(bd.ParentHash()))
}

// SignThis sets signature info in block data
func (bd *BlockData) SignThis(signer signature.Signature) error {
	sig, err := signer.Sign(bd.hash)
	if err != nil {
		return err
	}
	bd.alg = signer.Algorithm()
	bd.sign = sig
	return nil
}

// VerifyIntegrity verifies if block signature is valid
func (bd *BlockData) VerifyIntegrity() error {
	if bd.height == GenesisHeight {
		if !byteutils.Equal(GenesisHash, bd.hash) {
			return ErrInvalidBlockHash
		}
		return nil
	}
	for _, tx := range bd.transactions {
		if err := tx.VerifyIntegrity(bd.chainID); err != nil {
			return err
		}
	}

	wantedHash := HashBlockData(bd)
	if !byteutils.Equal(wantedHash, bd.hash) {
		return ErrInvalidBlockHash
	}

	return nil
}

// ExecuteOnParentBlock returns Block object with state after block execution
func (bd *BlockData) ExecuteOnParentBlock(parent *Block, txMap TxFactory) (*Block, error) {

	if parent.Height()+1 != bd.Height() {
		return nil, ErrInvalidBlockHeight
	}

	block, err := prepareExecution(bd, parent)
	if err != nil {
		return nil, err
	}
	if err := block.VerifyExecution(txMap); err != nil {
		return nil, err
	}
	return block, err
}

// GetExecutedBlock converts BlockData instance to an already executed Block instance
func (bd *BlockData) GetExecutedBlock(consensus Consensus, storage storage.Storage) (*Block, error) {
	var err error
	block := &Block{
		BlockData: bd,
		consensus: consensus,
	}
	if block.state, err = NewBlockState(block.consensus, storage); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to create new block state.")
		return nil, err
	}
	if err = block.state.LoadAccountsRoot(block.accsRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load accounts root.")
		return nil, err
	}
	if err = block.state.LoadTransactionsRoot(block.txsRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load transaction root.")
		return nil, err
	}
	if err = block.state.LoadUsageRoot(block.usageRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load usage root.")
		return nil, err
	}
	if err = block.state.LoadRecordsRoot(block.recordsRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load records root.")
		return nil, err
	}
	if err = block.state.LoadCertificationRoot(block.certificationRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load certification root.")
		return nil, err
	}

	ds, err := consensus.LoadConsensusState(block.dposRoot, storage)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load consensus state.")
		return nil, err
	}
	block.state.dposState = ds

	if err := block.state.LoadReservationQueue(block.reservationQueueHash); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load reservation queue.")
		return nil, err
	}
	block.storage = storage
	return block, nil
}

// prepareExecution by setting states and storage as those of parents
func prepareExecution(bd *BlockData, parent *Block) (*Block, error) {
	block, err := NewBlock(parent.ChainID(), bd.Coinbase(), parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to make new block for prepareExcution")
		return nil, err
	}

	block.BlockData = bd

	if err := block.BeginBatch(); err != nil {
		return nil, err
	}

	if err := block.SetMintDposState(parent); err != nil {
		block.RollBack()
		return nil, err
	}

	if err := block.Commit(); err != nil {
		return nil, err
	}

	return block, nil
}

// Block represents block with actual state tries
type Block struct {
	*BlockData
	storage   storage.Storage
	state     *BlockState
	consensus Consensus
	sealed    bool
}

func (block *Block) Consensus() Consensus {
	return block.consensus
}

// NewBlock initialize new block data
func NewBlock(chainID uint32, coinbase common.Address, parent *Block) (*Block, error) {
	state, err := parent.state.Clone()
	if err != nil {
		return nil, err
	}

	block := &Block{
		BlockData: &BlockData{
			BlockHeader: &BlockHeader{
				parentHash: parent.Hash(),
				coinbase:   coinbase,
				timestamp:  time.Now().Unix(),
				chainID:    chainID,
			},
			transactions: make(Transactions, 0),
			height:       parent.height + 1,
		},
		storage:   parent.storage,
		state:     state,
		consensus: parent.consensus,
		sealed:    false,
	}
	return block, nil
}

func (block *Block) Clone() (*Block, error) {

	bd, err := block.BlockData.Clone()
	if err != nil {
		return nil, err
	}

	state, err := block.state.Clone()
	if err != nil {
		return nil, err
	}

	return &Block{
		BlockData: bd,
		storage:   block.storage,
		state:     state,
		consensus: block.consensus,
		sealed:    block.sealed,
	}, nil
}

// State returns block state
func (block *Block) State() *BlockState {
	return block.state
}

// Storage returns storage used by block
func (block *Block) Storage() storage.Storage {
	return block.storage
}

// Sealed returns sealed
func (b *Block) Sealed() bool {
	return b.sealed
}

func (b *Block) SetSealed(sealed bool) {
	b.sealed = sealed
}

// Seal writes state root hashes and block hash in block header
func (block *Block) Seal() error {
	if block.sealed {
		return ErrBlockAlreadySealed
	}

	// all reserved tasks should have timestamps greater than block's timestamp
	head := block.state.PeekHeadReservedTask()
	if head != nil && head.Timestamp() < block.Timestamp() {
		return ErrReservedTaskNotProcessed
	}

	block.accsRoot = block.state.AccountsRoot()
	block.txsRoot = block.state.TransactionsRoot()
	block.usageRoot = block.state.UsageRoot()
	block.recordsRoot = block.state.RecordsRoot()
	block.certificationRoot = block.state.CertificationRoot()
	dposRoot, err := block.state.dposState.RootBytes()
	if err != nil {
		return err
	}
	block.dposRoot = dposRoot
	block.reservationQueueHash = block.state.ReservationQueueHash()

	hash := HashBlockData(block.BlockData)
	block.hash = hash
	block.sealed = true
	return nil
}

// HashBlockData returns hash of block
func HashBlockData(bd *BlockData) []byte {
	hasher := sha3.New256()

	hasher.Write(bd.ParentHash())
	hasher.Write(bd.Coinbase().Bytes())
	hasher.Write(bd.AccsRoot())
	hasher.Write(bd.TxsRoot())
	hasher.Write(bd.UsageRoot())
	hasher.Write(bd.RecordsRoot())
	hasher.Write(bd.CertificationRoot())
	hasher.Write(bd.DposRoot())
	hasher.Write(bd.ReservationQueueHash())
	hasher.Write(byteutils.FromInt64(bd.Timestamp()))
	hasher.Write(byteutils.FromUint32(bd.ChainID()))

	for _, tx := range bd.transactions {
		hasher.Write(tx.Hash())
	}

	return hasher.Sum(nil)
}

// ExecuteTransaction on given block state
func (block *Block) ExecuteTransaction(transaction *Transaction, txMap TxFactory) error {
	newTxFunc, ok := txMap[transaction.Type()]
	if !ok {
		return ErrInvalidTransactionType
	}

	tx, err := newTxFunc(transaction)
	if err != nil {
		return err
	}
	return tx.Execute(block)
}

// VerifyExecution executes txs in block and verify root hashes using block header
func (block *Block) VerifyExecution(txMap TxFactory) error {
	block.BeginBatch()

	if err := block.ExecuteReservedTasks(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Warn("Failed to execute reserved tasks.")
		block.RollBack()
		return err
	}

	if err := block.ExecuteAll(txMap); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to execute block transactions.")
		block.RollBack()
		return err
	}

	if err := block.VerifyState(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to verify block state.")
		block.RollBack()
		return err
	}

	block.Commit()

	return nil
}

// ExecuteAll executes all txs in block
func (block *Block) ExecuteAll(txMap TxFactory) error {
	block.BeginBatch()

	for _, transaction := range block.transactions {
		err := block.Execute(transaction, txMap)
		if err != nil {
			block.RollBack()
			return err
		}
	}

	block.Commit()

	return nil
}

// Execute executes a transaction.
func (block *Block) Execute(tx *Transaction, txMap TxFactory) error {
	err := block.state.checkNonce(tx)
	if err != nil {
		return err
	}

	if err := block.ExecuteTransaction(tx, txMap); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       block,
		}).Warn("Failed to execute a transaction.")
		return err
	}

	if err := block.state.AcceptTransaction(tx, block.Timestamp()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       block,
		}).Warn("Failed to accept a transaction.")
		return err
	}
	return nil
}

// ExecuteReservedTasks processes reserved tasks with timestamp before block's timestamp
func (block *Block) ExecuteReservedTasks() error {
	tasks := block.state.PopReservedTasks(block.Timestamp())
	for _, t := range tasks {
		if err := t.ExecuteOnState(block.state); err != nil {
			return err
		}
	}
	return nil
}

// AcceptTransaction adds tx in block state
func (block *Block) AcceptTransaction(tx *Transaction) error {
	if err := block.state.AcceptTransaction(tx, block.Timestamp()); err != nil {
		return err
	}
	block.transactions = append(block.transactions, tx)
	return nil
}

// VerifyState verifies block states comparing with root hashes in header
func (block *Block) VerifyState() error {
	if !byteutils.Equal(block.state.AccountsRoot(), block.AccsRoot()) {
		logging.Console().WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(block.state.AccountsRoot()),
			"header": byteutils.Bytes2Hex(block.AccsRoot()),
		}).Warn("Failed to verify accounts root.")
		return ErrInvalidBlockAccountsRoot
	}
	if !byteutils.Equal(block.state.TransactionsRoot(), block.TxsRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(block.state.TransactionsRoot()),
			"header": byteutils.Bytes2Hex(block.TxsRoot()),
		}).Warn("Failed to verify transactions root.")
		return ErrInvalidBlockTxsRoot
	}
	if !byteutils.Equal(block.state.UsageRoot(), block.UsageRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(block.state.UsageRoot()),
			"header": byteutils.Bytes2Hex(block.UsageRoot()),
		}).Warn("Failed to verify usage root.")
		return ErrInvalidBlockUsageRoot
	}
	if !byteutils.Equal(block.state.RecordsRoot(), block.RecordsRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(block.state.RecordsRoot()),
			"header": byteutils.Bytes2Hex(block.RecordsRoot()),
		}).Warn("Failed to verify records root.")
		return ErrInvalidBlockRecordsRoot
	}
	if !byteutils.Equal(block.state.CertificationRoot(), block.CertificationRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(block.state.CertificationRoot()),
			"header": byteutils.Bytes2Hex(block.CertificationRoot()),
		}).Warn("Failed to verify certification root.")
		return ErrInvalidBlockCertificationRoot
	}
	dposRoot, err := block.state.DposState().RootBytes()
	if err != nil {
		return err
	}
	if !byteutils.Equal(dposRoot, block.DposRoot()) {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get state of candidate root.")
		return err
	}
	if !byteutils.Equal(block.state.ReservationQueueHash(), block.ReservationQueueHash()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(block.state.ReservationQueueHash()),
			"header": byteutils.Bytes2Hex(block.ReservationQueueHash()),
		}).Warn("Failed to verify reservation queue hash.")
		return ErrInvalidBlockReservationQueueHash
	}
	return nil
}

// SignThis sets signature info in block
func (block *Block) SignThis(signer signature.Signature) error {
	if !block.Sealed() {
		return ErrBlockNotSealed
	}

	return block.BlockData.SignThis(signer)
}

func (block *Block) SetMintDposState(parent *Block) error {
	d := block.consensus
	return d.SetMintDynastyState(block.timestamp, parent, block)
}

// BeginBatch makes block state update possible
func (block *Block) BeginBatch() error {
	return block.state.BeginBatch()
}

// RollBack rolls back block state batch updates
func (block *Block) RollBack() error {
	return block.state.RollBack()
}

// Commit saves batch updates to storage
func (block *Block) Commit() error {
	return block.state.Commit()
}

// GetBlockData returns data part of block
func (block *Block) GetBlockData() *BlockData {
	return block.BlockData
}

// EmitTxExecutionEvent emits events of txs in the block
func (block *Block) EmitTxExecutionEvent(emitter *EventEmitter) {
	for _, tx := range block.Transactions() {
		event := &Event{
			Topic: TopicTransactionExecutionResult,
			Data:  tx.String(),
		}
		emitter.Trigger(event)
	}
}

// BytesToBlockData unmarshals proto bytes to BlockData.
func BytesToBlockData(bytes []byte) (*BlockData, error) {
	pb := new(corepb.Block)
	if err := proto.Unmarshal(bytes, pb); err != nil {
		return nil, err
	}
	bd := new(BlockData)
	if err := bd.FromProto(pb); err != nil {
		return nil, err
	}
	return bd, nil
}
