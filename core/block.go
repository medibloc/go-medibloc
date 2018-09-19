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

	accStateRoot []byte
	txStateRoot  []byte
	dposRoot     []byte

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
		Hash:         b.hash,
		ParentHash:   b.parentHash,
		Coinbase:     b.coinbase.Bytes(),
		Reward:       reward,
		Supply:       supply,
		Timestamp:    b.timestamp,
		ChainId:      b.chainID,
		Alg:          uint32(b.alg),
		Sign:         b.sign,
		AccStateRoot: b.accStateRoot,
		TxStateRoot:  b.txStateRoot,
		DposRoot:     b.dposRoot,
	}, nil
}

// FromProto converts corepb.BlockHeader to BlockHeader
func (b *BlockHeader) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.BlockHeader); ok {
		b.hash = msg.Hash
		b.parentHash = msg.ParentHash
		b.accStateRoot = msg.AccStateRoot
		b.txStateRoot = msg.TxStateRoot
		b.dposRoot = msg.DposRoot
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

//AccStateRoot returns block header's accStateRoot
func (b *BlockHeader) AccStateRoot() []byte {
	return b.accStateRoot
}

//SetAccStateRoot set block header's accStateRoot
func (b *BlockHeader) SetAccStateRoot(accStateRoot []byte) {
	b.accStateRoot = accStateRoot
}

//TxStateRoot returns block header's txsRoot
func (b *BlockHeader) TxStateRoot() []byte {
	return b.txStateRoot
}

//SetTxStateRoot set block header's txsRoot
func (b *BlockHeader) SetTxStateRoot(txStateRoot []byte) {
	b.txStateRoot = txStateRoot
}

//DposRoot returns block header's dposRoot
func (b *BlockHeader) DposRoot() []byte {
	return b.dposRoot
}

//SetDposRoot set block header's dposRoot
func (b *BlockHeader) SetDposRoot(dposRoot []byte) {
	b.dposRoot = dposRoot
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
	transactions []*Transaction
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

		bd.transactions = make([]*Transaction, len(msg.Transactions))
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
func (bd *BlockData) Transactions() []*Transaction {
	return bd.transactions
}

// SetTransactions sets transactions TO BE REMOVED: For test without block pool
func (bd *BlockData) SetTransactions(txs []*Transaction) error {
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
	block.Storage().EnableBatch()
	if err := block.VerifyExecution(parent, txMap); err != nil {
		block.Storage().DisableBatch()
		return nil, err
	}
	err = block.Storage().Flush()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to flush to storage.")
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

	if err = block.state.LoadAccountState(block.accStateRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load accounts root.")
		return nil, err
	}
	if err = block.state.LoadTransactionState(block.txStateRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load transaction root.")
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

	block.storage = storage
	return block, nil
}

// prepareExecution by setting states and storage as those of parents
func prepareExecution(bd *BlockData, parent *Block) (*Block, error) {
	block, err := NewBlock(parent.ChainID(), bd.Coinbase(), parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to make new block for prepareExecution")
		return nil, err
	}

	block.BlockData = bd

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
			transactions: make([]*Transaction, 0),
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
	var err error
	if b.sealed {
		return ErrBlockAlreadySealed
	}

	b.reward = b.state.Reward()
	b.supply = b.state.Supply()
	b.accStateRoot, err = b.state.AccountsRoot()
	if err != nil {
		return err
	}
	b.txStateRoot, err = b.state.TxsRoot()
	if err != nil {
		return err
	}
	b.dposRoot, err = b.state.dposState.RootBytes()
	if err != nil {
		return err
	}

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
	hasher.Write(bd.AccStateRoot())
	hasher.Write(bd.TxStateRoot())
	hasher.Write(bd.DposRoot())
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

	if transaction.Timestamp() < b.Timestamp()-TxDelayLimit {
		return ErrTooOldTransaction
	}

	newTxFunc, ok := txMap[transaction.TxType()]
	if !ok {
		return ErrInvalidTransactionType
	}

	tx, err := newTxFunc(transaction)
	if err != nil {
		return err
	}

	payer, err := transaction.recoverPayer()
	if err == ErrPayerSignatureNotExist {
		payer = transaction.from
	} else if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to recover a payer address.")
		return err
	}

	err = b.regenerateBandwidth(payer)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to regenerate bandwidth.")
		return err
	}

	err = b.updateUnstaking(transaction)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to update staking.")
		return err
	}

	err = tx.Execute(b)
	if err != nil {
		return err
	}

	usage, err := tx.Bandwidth()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get bandwidth of a transaction.")
		return err
	}

	err = b.consumeBandwidth(payer, usage)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to update bandwidth.")
		return err
	}
	return nil
}

func (b *Block) updateUnstaking(tx *Transaction) error {
	addr := tx.From()
	acc, err := b.State().GetAccount(addr)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get account.")
		return err
	}

	if acc.LastUnstakingTs == 0 {
		return nil
	}

	elapsed := b.Timestamp() - acc.LastUnstakingTs
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

	err = b.State().PutAccount(acc)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to put account.")
		return err
	}

	return nil
}

func (b *Block) consumeBandwidth(addr common.Address, usage *util.Uint128) error {
	acc, err := b.State().GetAccount(addr)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get account.")
		return err
	}

	avail, err := acc.Vesting.Sub(acc.Bandwidth)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to calculate available usage.")
		return err
	}
	if avail.Cmp(usage) < 0 {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Bandwidth limit exceeded.")
		return ErrBandwidthLimitExceeded
	}

	updated, err := acc.Bandwidth.Add(usage)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to calculate bandwidth.")
		return err
	}
	acc.Bandwidth = updated
	acc.LastBandwidthTs = b.Timestamp()

	err = b.State().PutAccount(acc)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to put account.")
		return err
	}
	return nil
}

func (b *Block) regenerateBandwidth(addr common.Address) error {
	curTs := b.Timestamp()

	acc, err := b.State().GetAccount(addr)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":  err,
			"addr": addr.Hex(),
		}).Warn("Failed to get account.")
		return err
	}
	vesting, bandwidth, lastTs := acc.Vesting, acc.Bandwidth, acc.LastBandwidthTs

	curBandwidth, err := currentBandwidth(vesting, bandwidth, lastTs, curTs)
	if err != nil {
		return err
	}

	acc.Bandwidth = curBandwidth
	acc.LastBandwidthTs = curTs
	err = b.State().PutAccount(acc)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to put account.")
		return err
	}
	return nil
}

func currentBandwidth(vesting, bandwidth *util.Uint128, lastTs, curTs int64) (*util.Uint128, error) {
	elapsed := curTs - lastTs
	if time.Duration(elapsed)*time.Second >= BandwidthRegenerateDuration {
		return util.NewUint128(), nil
	}

	if elapsed < 0 {
		return nil, ErrInvalidTimestamp
	}

	if elapsed == 0 {
		return bandwidth.DeepCopy(), nil
	}

	mul := util.NewUint128FromUint(uint64(elapsed))
	div := util.NewUint128FromUint(uint64(BandwidthRegenerateDuration / time.Second))
	v1, err := vesting.Mul(mul)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to multiply uint128.")
		return nil, err
	}
	regen, err := v1.Div(div)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to divide uint128.")
		return nil, err
	}

	if regen.Cmp(bandwidth) >= 0 {
		return util.NewUint128(), nil
	}
	cur, err := bandwidth.Sub(regen)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to subtract uint128.")
		return nil, err
	}
	return cur, nil
}

// VerifyExecution executes txs in block and verify root hashes using block header
func (b *Block) VerifyExecution(parent *Block, txMap TxFactory) error {
	b.BeginBatch()

	if err := b.SetMintDynastyState(parent); err != nil {
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

	if err := b.state.acceptTransaction(tx, b.Timestamp()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
		}).Warn("Failed to accept a transaction.")
		return err
	}
	return nil
}

//PayReward add reward to coinbase and update reward and supply
func (b *Block) PayReward(coinbase common.Address, parentSupply *util.Uint128) error {
	reward, err := calcMintReward(parentSupply)
	if err != nil {
		return err
	}

	acc, err := b.state.GetAccount(coinbase)
	if err != nil {
		return err
	}
	acc.Balance, err = acc.Balance.Add(reward)
	if err != nil {
		return err
	}
	err = b.state.PutAccount(acc)
	if err != nil {
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
	if err := b.state.acceptTransaction(tx, b.Timestamp()); err != nil {
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

	accRoot, err := b.state.AccountsRoot()
	if err != nil {
		return err
	}
	if !byteutils.Equal(accRoot, b.AccStateRoot()) {
		logging.Console().WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(accRoot),
			"header": byteutils.Bytes2Hex(b.AccStateRoot()),
		}).Warn("Failed to verify accounts root.")
		return ErrInvalidBlockAccountsRoot
	}

	txsRoot, err := b.state.TxsRoot()
	if err != nil {
		return err
	}
	if !byteutils.Equal(txsRoot, b.TxStateRoot()) {
		logging.WithFields(logrus.Fields{
			"state":  byteutils.Bytes2Hex(txsRoot),
			"header": byteutils.Bytes2Hex(b.TxStateRoot()),
		}).Warn("Failed to verify transactions root.")
		return ErrInvalidBlockTxsRoot
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
	return nil
}

// SignThis sets signature info in block
func (b *Block) SignThis(signer signature.Signature) error {
	if !b.Sealed() {
		return ErrBlockNotSealed
	}

	return b.BlockData.SignThis(signer)
}

//SetMintDynastyState set mint dys
func (b *Block) SetMintDynastyState(parent *Block) error {
	return b.state.dposState.SetMintDynastyState(b.timestamp, parent, b.consensus.DynastySize())
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
			Data:  byteutils.Bytes2Hex(tx.Hash()),
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

	roundDecimal, err := util.NewUint128FromString(DecimalCount)
	if err != nil {
		return nil, err
	}
	reward, err = reward.Div(roundDecimal)
	if err != nil {
		return nil, err
	}
	reward, err = reward.Mul(roundDecimal)
	if err != nil {
		return nil, err
	}
	return reward, nil
}
