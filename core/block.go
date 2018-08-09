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
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
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
	reward    *util.Uint128
	supply    *util.Uint128
	timestamp int64
	chainID   uint32

	alg  algorithm.Algorithm
	sign []byte
}

// ToProto converts BlockHeader to corepb.BlockHeader
func (b *BlockHeader) ToProto() (proto.Message, error) {
	reward, err := b.reward.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	supply, err := b.supply.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	return &corepb.BlockHeader{
		Hash:                 b.hash,
		ParentHash:           b.parentHash,
		Coinbase:             b.coinbase.Bytes(),
		Reward:               reward,
		Supply:               supply,
		Timestamp:            b.timestamp,
		ChainId:              b.chainID,
		Alg:                  uint32(b.alg),
		Sign:                 b.sign,
		AccsRoot:             b.accsRoot,
		TxsRoot:              b.txsRoot,
		UsageRoot:            b.usageRoot,
		RecordsRoot:          b.recordsRoot,
		CertificationRoot:    b.certificationRoot,
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
		reward, err := util.NewUint128FromFixedSizeByteSlice(msg.Reward)
		if err != nil {
			return err
		}
		b.reward = reward
		supply, err := util.NewUint128FromFixedSizeByteSlice(msg.Supply)
		if err != nil {
			return err
		}
		b.supply = supply
		b.timestamp = msg.Timestamp
		b.chainID = msg.ChainId
		b.alg = algorithm.Algorithm(msg.Alg)
		b.sign = msg.Sign
		return nil
	}
	return ErrInvalidProtoToBlockHeader
}

//Hash returns block header's hash
func (b *BlockHeader) Hash() []byte {
	return b.hash
}

//SetHash set block header's hash
func (b *BlockHeader) SetHash(hash []byte) {
	b.hash = hash
}

//ParentHash returns block header's parent hash
func (b *BlockHeader) ParentHash() []byte {
	return b.parentHash
}

//SetParentHash set block header's parent hash
func (b *BlockHeader) SetParentHash(parentHash []byte) {
	b.parentHash = parentHash
}

//AccsRoot returns block header's accsRoot
func (b *BlockHeader) AccsRoot() []byte {
	return b.accsRoot
}

//SetAccsRoot set block header's accsRoot
func (b *BlockHeader) SetAccsRoot(accsRoot []byte) {
	b.accsRoot = accsRoot
}

//TxsRoot returns block header's txsRoot
func (b *BlockHeader) TxsRoot() []byte {
	return b.txsRoot
}

//SetTxsRoot set block header's txsRoot
func (b *BlockHeader) SetTxsRoot(txsRoot []byte) {
	b.txsRoot = txsRoot
}

//UsageRoot returns block header's usageRoot
func (b *BlockHeader) UsageRoot() []byte {
	return b.usageRoot
}

//SetUsageRoot set block header's usageRoot
func (b *BlockHeader) SetUsageRoot(usageRoot []byte) {
	b.usageRoot = usageRoot
}

//RecordsRoot returns block header's recordRoot
func (b *BlockHeader) RecordsRoot() []byte {
	return b.recordsRoot
}

//SetRecordsRoot set block header's recordRoot
func (b *BlockHeader) SetRecordsRoot(recordsRoot []byte) {
	b.recordsRoot = recordsRoot
}

//CertificationRoot returns block header's CertificationRoot
func (b *BlockHeader) CertificationRoot() []byte {
	return b.certificationRoot
}

//SetCertificationRoot set block header's CertificationRoot
func (b *BlockHeader) SetCertificationRoot(certificationRoot []byte) {
	b.certificationRoot = certificationRoot
}

//DposRoot returns block header's dposRoot
func (b *BlockHeader) DposRoot() []byte {
	return b.dposRoot
}

//SetDposRoot set block header's dposRoot
func (b *BlockHeader) SetDposRoot(dposRoot []byte) {
	b.dposRoot = dposRoot
}

//ReservationQueueHash returns block header's reservationQueHash
func (b *BlockHeader) ReservationQueueHash() []byte {
	return b.reservationQueueHash
}

//SetReservationQueueHash set block header's reservationQueHash
func (b *BlockHeader) SetReservationQueueHash(reservationQueueHash []byte) {
	b.reservationQueueHash = reservationQueueHash
}

//Coinbase returns coinbase
func (b *BlockHeader) Coinbase() common.Address {
	return b.coinbase
}

//SetCoinbase set coinbase
func (b *BlockHeader) SetCoinbase(coinbase common.Address) {
	b.coinbase = coinbase
}

//Reward returns reward
func (b *BlockHeader) Reward() *util.Uint128 {
	return b.reward
}

//SetReward sets reward
func (b *BlockHeader) SetReward(reward *util.Uint128) {
	b.reward = reward
}

//Supply returns supply
func (b *BlockHeader) Supply() *util.Uint128 {
	return b.supply.DeepCopy()
}

//SetSupply sets supply
func (b *BlockHeader) SetSupply(supply *util.Uint128) {
	b.supply = supply
}

//Timestamp returns timestamp of block
func (b *BlockHeader) Timestamp() int64 {
	return b.timestamp
}

//SetTimestamp sets timestamp of block
func (b *BlockHeader) SetTimestamp(timestamp int64) {
	b.timestamp = timestamp
}

//ChainID returns chainID
func (b *BlockHeader) ChainID() uint32 {
	return b.chainID
}

//SetChainID sets chainID
func (b *BlockHeader) SetChainID(chainID uint32) {
	b.chainID = chainID
}

//Alg returns signing algorithm
func (b *BlockHeader) Alg() algorithm.Algorithm {
	return b.alg
}

//SetAlg sets signing algorithm
func (b *BlockHeader) SetAlg(alg algorithm.Algorithm) {
	b.alg = alg
}

//Sign returns sign
func (b *BlockHeader) Sign() []byte {
	return b.sign
}

//SetSign sets sign
func (b *BlockHeader) SetSign(sign []byte) {
	b.sign = sign
}

//Proposer returns miner address from block sign
func (b *BlockHeader) Proposer() (common.Address, error) {
	if b.sign == nil {
		return common.Address{}, ErrBlockSignatureNotExist
	}
	msg := b.hash

	sig, err := crypto.NewSignature(b.alg)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"algorithm": b.alg,
		}).Debug("Invalid sign algorithm.")
		return common.Address{}, err
	}

	pubKey, err := sig.RecoverPublic(msg, b.sign)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":    err,
			"plain":  byteutils.Bytes2Hex(b.hash),
			"cipher": byteutils.Bytes2Hex(b.sign),
		}).Debug("Failed to recover public key from cipher text.")
		return common.Address{}, err
	}

	return common.PublicKeyToAddress(pubKey)
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

//Clone copy block data
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
	proposer, _ := bd.Proposer()
	return fmt.Sprintf("<height:%v, hash:%v, parent_hash:%v, coinbase:%v, reward:%v, supply:%v, timestamp:%v, proposer:%v>",
		bd.Height(),
		byteutils.Bytes2Hex(bd.Hash()),
		byteutils.Bytes2Hex(bd.ParentHash()),
		byteutils.Bytes2Hex(bd.Coinbase().Bytes()),
		bd.Reward().String(),
		bd.Supply().String(),
		bd.Timestamp(),
		proposer.Hex(),
	)
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

	block.state.reward = bd.Reward()
	block.state.supply = bd.Supply()

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

//Consensus returns block's consensus
func (b *Block) Consensus() Consensus {
	return b.consensus
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
				reward:     parent.Reward(),
				supply:     parent.Supply(),
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

//Clone clone block
func (b *Block) Clone() (*Block, error) {

	bd, err := b.BlockData.Clone()
	if err != nil {
		return nil, err
	}

	state, err := b.state.Clone()
	if err != nil {
		return nil, err
	}

	return &Block{
		BlockData: bd,
		storage:   b.storage,
		state:     state,
		consensus: b.consensus,
		sealed:    b.sealed,
	}, nil
}

// State returns block state
func (b *Block) State() *BlockState {
	return b.state
}

// Storage returns storage used by block
func (b *Block) Storage() storage.Storage {
	return b.storage
}

// Sealed returns sealed
func (b *Block) Sealed() bool {
	return b.sealed
}

//SetSealed set sealed
func (b *Block) SetSealed(sealed bool) {
	b.sealed = sealed
}

// Seal writes state root hashes and block hash in block header
func (b *Block) Seal() error {
	if b.sealed {
		return ErrBlockAlreadySealed
	}

	// all reserved tasks should have timestamps greater than block's timestamp
	head := b.state.PeekHeadReservedTask()
	if head != nil && head.Timestamp() < b.Timestamp() {
		return ErrReservedTaskNotProcessed
	}

	b.reward = b.state.Reward()
	b.supply = b.state.Supply()
	b.accsRoot = b.state.AccountsRoot()
	b.txsRoot = b.state.TransactionsRoot()
	b.usageRoot = b.state.UsageRoot()
	b.recordsRoot = b.state.RecordsRoot()
	b.certificationRoot = b.state.CertificationRoot()
	dposRoot, err := b.state.dposState.RootBytes()
	if err != nil {
		return err
	}
	b.dposRoot = dposRoot
	b.reservationQueueHash = b.state.ReservationQueueHash()

	hash := HashBlockData(b.BlockData)
	b.hash = hash
	b.sealed = true
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
func (b *Block) ExecuteTransaction(transaction *Transaction, txMap TxFactory) error {
	err := b.state.checkNonce(transaction)
	if err != nil {
		return err
	}

	newTxFunc, ok := txMap[transaction.Type()]
	if !ok {
		return ErrInvalidTransactionType
	}

	tx, err := newTxFunc(transaction)
	if err != nil {
		return err
	}
	return tx.Execute(b)
}

// VerifyExecution executes txs in block and verify root hashes using block header
func (b *Block) VerifyExecution(txMap TxFactory) error {
	b.BeginBatch()

	if err := b.ExecuteReservedTasks(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Warn("Failed to execute reserved tasks.")
		b.RollBack()
		return err
	}

	if err := b.ExecuteAll(txMap); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to execute block transactions.")
		b.RollBack()
		return err
	}

	if err := b.PayReward(b.coinbase, b.State().Supply()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to pay block reward.")
		b.RollBack()
		return err
	}

	b.Commit()

	if err := b.VerifyState(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to verify block state.")
		b.RollBack()
		return err
	}

	return nil
}

// ExecuteAll executes all txs in block
func (b *Block) ExecuteAll(txMap TxFactory) error {

	for _, transaction := range b.transactions {
		err := b.Execute(transaction, txMap)
		if err != nil {
			b.RollBack()
			return err
		}
	}

	return nil
}

// Execute executes a transaction.
func (b *Block) Execute(tx *Transaction, txMap TxFactory) error {

	if err := b.ExecuteTransaction(tx, txMap); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
		}).Warn("Failed to execute a transaction.")
		return err
	}

	if err := b.state.AcceptTransaction(tx, b.Timestamp()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
		}).Warn("Failed to accept a transaction.")
		return err
	}
	return nil
}

// ExecuteReservedTasks processes reserved tasks with timestamp before block's timestamp
func (b *Block) ExecuteReservedTasks() error {
	tasks := b.state.PopReservedTasks(b.Timestamp())
	for _, t := range tasks {
		if err := t.ExecuteOnState(b.state); err != nil {
			return err
		}
	}
	return nil
}

//PayReward add reward to coinbase and update reward and supply
func (b *Block) PayReward(coinbase common.Address, parentSupply *util.Uint128) error {
	reward, err := calcMintReward(parentSupply)
	if err != nil {
		return err
	}

	if err := b.state.accState.AddBalance(coinbase, reward); err != nil {
		return err
	}

	supply, err := parentSupply.Add(reward)
	if err != nil {
		return err
	}

	b.state.reward = reward
	b.state.supply = supply

	return nil
}

// AcceptTransaction adds tx in block state
func (b *Block) AcceptTransaction(tx *Transaction) error {
	if err := b.state.AcceptTransaction(tx, b.Timestamp()); err != nil {
		return err
	}
	b.transactions = append(b.transactions, tx)
	return nil
}

// VerifyState verifies block states comparing with root hashes in header
func (b *Block) VerifyState() error {
	if b.state.Reward().Cmp(b.Reward()) != 0 {
		logging.Console().WithFields(logrus.Fields{
			"state":  b.state.Reward(),
			"header": b.Reward(),
		}).Warn("Failed to verify reward.")
		return ErrInvalidBlockReward
	}
	if b.state.Supply().Cmp(b.Supply()) != 0 {
		logging.Console().WithFields(logrus.Fields{
			"state":  b.state.Supply(),
			"header": b.Supply(),
		}).Warn("Failed to verify supply.")
		return ErrInvalidBlockSupply
	}

	if !byteutils.Equal(b.state.AccountsRoot(), b.AccsRoot()) {
		logging.Console().WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(b.state.AccountsRoot()),
			"header": byteutils.Bytes2Hex(b.AccsRoot()),
		}).Warn("Failed to verify accounts root.")
		return ErrInvalidBlockAccountsRoot
	}
	if !byteutils.Equal(b.state.TransactionsRoot(), b.TxsRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(b.state.TransactionsRoot()),
			"header": byteutils.Bytes2Hex(b.TxsRoot()),
		}).Warn("Failed to verify transactions root.")
		return ErrInvalidBlockTxsRoot
	}
	if !byteutils.Equal(b.state.UsageRoot(), b.UsageRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(b.state.UsageRoot()),
			"header": byteutils.Bytes2Hex(b.UsageRoot()),
		}).Warn("Failed to verify usage root.")
		return ErrInvalidBlockUsageRoot
	}
	if !byteutils.Equal(b.state.RecordsRoot(), b.RecordsRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(b.state.RecordsRoot()),
			"header": byteutils.Bytes2Hex(b.RecordsRoot()),
		}).Warn("Failed to verify records root.")
		return ErrInvalidBlockRecordsRoot
	}
	if !byteutils.Equal(b.state.CertificationRoot(), b.CertificationRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(b.state.CertificationRoot()),
			"header": byteutils.Bytes2Hex(b.CertificationRoot()),
		}).Warn("Failed to verify certification root.")
		return ErrInvalidBlockCertificationRoot
	}
	dposRoot, err := b.state.DposState().RootBytes()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get dpos state's root bytes.")
		return err
	}
	if !byteutils.Equal(dposRoot, b.DposRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(dposRoot),
			"header": byteutils.Bytes2Hex(b.DposRoot()),
		}).Warn("Failed to get state of candidate root.")
		return ErrInvalidBlockDposRoot
	}
	if !byteutils.Equal(b.state.ReservationQueueHash(), b.ReservationQueueHash()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(b.state.ReservationQueueHash()),
			"header": byteutils.Bytes2Hex(b.ReservationQueueHash()),
		}).Warn("Failed to verify reservation queue hash.")
		return ErrInvalidBlockReservationQueueHash
	}
	return nil
}

// SignThis sets signature info in block
func (b *Block) SignThis(signer signature.Signature) error {
	if !b.Sealed() {
		return ErrBlockNotSealed
	}

	return b.BlockData.SignThis(signer)
}

//SetMintDposState set mint dys
func (b *Block) SetMintDposState(parent *Block) error {
	d := b.consensus
	return d.SetMintDynastyState(b.timestamp, parent, b)
}

// BeginBatch makes block state update possible
func (b *Block) BeginBatch() error {
	return b.state.BeginBatch()
}

// RollBack rolls back block state batch updates
func (b *Block) RollBack() error {
	return b.state.RollBack()
}

// Commit saves batch updates to storage
func (b *Block) Commit() error {
	return b.state.Commit()
}

// GetBlockData returns data part of block
func (b *Block) GetBlockData() *BlockData {
	return b.BlockData
}

// EmitTxExecutionEvent emits events of txs in the block
func (b *Block) EmitTxExecutionEvent(emitter *EventEmitter) {
	for _, tx := range b.Transactions() {
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

//calcMintReward returns calculated block produce reward
func calcMintReward(parentSupply *util.Uint128) (*util.Uint128, error) {
	rateNum, err := util.NewUint128FromString(rateNum)
	if err != nil {
		return nil, err
	}
	rateDecimal, err := util.NewUint128FromString(rateDecimal)
	if err != nil {
		return nil, err
	}
	tempReward, err := rateNum.Mul(parentSupply)
	if err != nil {
		return nil, err
	}
	reward, err := tempReward.Div(rateDecimal)
	if err != nil {
		return nil, err
	}

	return reward, nil

}
