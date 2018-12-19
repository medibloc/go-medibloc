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
	"math/big"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/hash"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
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

	sign []byte

	cpuPrice *util.Uint128
	cpuUsage uint64
	netPrice *util.Uint128
	netUsage uint64
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

	cpuPrice, err := b.cpuPrice.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	netPrice, err := b.netPrice.ToFixedSizeByteSlice()
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
		Sign:         b.sign,
		AccStateRoot: b.accStateRoot,
		TxStateRoot:  b.txStateRoot,
		DposRoot:     b.dposRoot,
		CpuPrice:     cpuPrice,
		CpuUsage:     b.cpuUsage,
		NetPrice:     netPrice,
		NetUsage:     b.netUsage,
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
		err := b.coinbase.FromBytes(msg.Coinbase)
		if err != nil {
			return err
		}
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
		b.sign = msg.Sign

		cpuPrice, err := util.NewUint128FromFixedSizeByteSlice(msg.CpuPrice)
		if err != nil {
			return err
		}

		netPrice, err := util.NewUint128FromFixedSizeByteSlice(msg.NetPrice)
		if err != nil {
			return err
		}

		b.cpuPrice = cpuPrice
		b.cpuUsage = msg.CpuUsage
		b.netPrice = netPrice
		b.netUsage = msg.NetUsage
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

//CPUPrice returns cpuPrice
func (b *BlockHeader) CPUPrice() *util.Uint128 {
	return b.cpuPrice
}

//SetCPUPrice sets cpuPrice
func (b *BlockHeader) SetCPUPrice(cpuPrice *util.Uint128) {
	b.cpuPrice = cpuPrice
}

//NetPrice returns netPrice
func (b *BlockHeader) NetPrice() *util.Uint128 {
	return b.netPrice
}

//SetNetPrice sets netPrice
func (b *BlockHeader) SetNetPrice(netPrice *util.Uint128) {
	b.netPrice = netPrice
}

//Sign returns sign
func (b *BlockHeader) Sign() []byte {
	return b.sign
}

//SetSign sets sign
func (b *BlockHeader) SetSign(sign []byte) {
	b.sign = sign
}

// CPUUsage returns cpuUsage
func (b *BlockHeader) CPUUsage() uint64 {
	return b.cpuUsage
}

// NetUsage returns netUsage
func (b *BlockHeader) NetUsage() uint64 {
	return b.netUsage
}

//Proposer returns proposer address from block sign
func (b *BlockHeader) Proposer() (common.Address, error) {
	if b.sign == nil {
		return common.Address{}, ErrBlockSignatureNotExist
	}
	msg := b.hash

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"algorithm": algorithm.SECP256K1,
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

//ToBytes convert block data to byte slice
func (bd *BlockData) ToBytes() ([]byte, error) {
	pb, err := bd.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pb)
}

//FromBytes convert byte slice to
func (bd *BlockData) FromBytes(bytes []byte) error {
	pb := new(corepb.Block)
	if err := proto.Unmarshal(bytes, pb); err != nil {
		return err
	}
	if err := bd.FromProto(pb); err != nil {
		return err
	}
	return nil
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
	return fmt.Sprintf("<height:%v, hash:%v, parent_hash:%v, coinbase:%v, reward:%v, supply:%v, timestamp:%v, "+
		"proposer:%v, cpuPrice:%v, cpuBandwidth:%v, netPrice:%v, netBandwidth:%v>",
		bd.Height(),
		byteutils.Bytes2Hex(bd.Hash()),
		byteutils.Bytes2Hex(bd.ParentHash()),
		byteutils.Bytes2Hex(bd.Coinbase().Bytes()),
		bd.Reward().String(),
		bd.Supply().String(),
		bd.Timestamp(),
		proposer.Hex(),
		bd.CPUPrice(),
		bd.CPUUsage(),
		bd.NetPrice(),
		bd.NetUsage(),
	)
}

// SignThis sets signature info in block data
func (bd *BlockData) SignThis(signer signature.Signature) error {
	sig, err := signer.Sign(bd.hash)
	if err != nil {
		return err
	}
	bd.sign = sig
	return nil
}

// VerifyIntegrity verifies if block signature is valid
func (bd *BlockData) VerifyIntegrity() error {
	if bd.height == GenesisHeight {
		//if !byteutils.Equal(GenesisHash, bd.hash) {
		//	return ErrInvalidBlockHash
		//}
		return nil
	}
	for _, tx := range bd.transactions {
		if err := tx.VerifyIntegrity(bd.chainID); err != nil {
			return err
		}
	}

	wantedHash, err := HashBlockData(bd)
	if err != nil {
		return err
	}
	if !byteutils.Equal(wantedHash, bd.hash) {
		return ErrInvalidBlockHash
	}

	return nil
}

// ExecuteOnParentBlock returns Block object with state after block execution
func (bd *BlockData) ExecuteOnParentBlock(parent *Block, consensus Consensus, txMap TxFactory) (*Block, error) {
	// Prepare Execution
	block, err := parent.Child()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to make child block for execution on parent block")
		return nil, err
	}
	block.BlockData = bd
	err = block.Prepare()
	if err != nil {
		return nil, err
	}

	if err := bd.verifyBandwidthUsage(); err != nil {
		return nil, err
	}

	if err := block.VerifyExecution(parent, consensus, txMap); err != nil {
		return nil, err
	}
	err = block.Flush()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to flush state")
	}
	return block, err
}

// GetExecutedBlock converts BlockData instance to an already executed Block instance
func (bd *BlockData) GetExecutedBlock(consensus Consensus, storage storage.Storage) (*Block, error) {
	var err error
	block := &Block{
		BlockData: bd,
	}
	if block.state, err = newStates(consensus, storage); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to create new block state.")
		return nil, err
	}

	block.state.reward = bd.Reward()
	block.state.supply = bd.Supply()

	block.state.cpuPrice = bd.cpuPrice
	block.state.cpuUsage = bd.cpuUsage
	block.state.netPrice = bd.netPrice
	block.state.netUsage = bd.netUsage

	if err = block.state.loadAccountState(block.accStateRoot); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to load accounts root.")
		return nil, err
	}
	if err = block.state.loadTransactionState(block.txStateRoot); err != nil {
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

func (bd *BlockData) verifyBandwidthUsage() error {
	if bd.CPUUsage() > CPULimit {
		return ErrInvalidCPUUsage
	}

	if bd.NetUsage() > NetLimit {
		return ErrInvalidNetUsage
	}

	if err := bd.verifyTotalBandwidth(); err != nil {
		return err
	}
	return nil
}

func (bd *BlockData) verifyTotalBandwidth() error {
	cpuUsage := uint64(0)
	netUsage := uint64(0)
	for _, tx := range bd.Transactions() {
		cpuUsage = cpuUsage + tx.receipt.cpuUsage
		netUsage = netUsage + tx.receipt.netUsage
	}

	if cpuUsage != bd.cpuUsage {
		return ErrWrongCPUUsage
	}
	if netUsage != bd.netUsage {
		return ErrWrongNetUsage
	}
	return nil
}

// EmitTxExecutionEvent emits events of txs in the block
func (bd *BlockData) EmitTxExecutionEvent(emitter *EventEmitter) {
	for _, tx := range bd.Transactions() {
		tx.TriggerEvent(emitter, TopicTransactionExecutionResult)
		tx.TriggerAccEvent(emitter, TypeAccountTransactionExecution)
	}
}

// EmitBlockEvent emits block related event
func (bd *BlockData) EmitBlockEvent(emitter *EventEmitter, eTopic string) {
	event := &Event{
		Topic: eTopic,
		Data:  byteutils.Bytes2Hex(bd.Hash()),
		Type:  "",
	}
	emitter.Trigger(event)

}

// Block represents block with actual state tries
type Block struct {
	*BlockData
	storage storage.Storage
	state   *BlockState
	sealed  bool
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
		sealed:    b.sealed,
	}, nil
}

//Child return initial child block for verifying or making block
func (b *Block) Child() (*Block, error) {
	state, err := b.state.Clone()
	if err != nil {
		return nil, err
	}
	state.cpuUsage = 0
	state.netUsage = 0

	state.cpuPrice, err = calcCPUPrice(b)
	if err != nil {
		return nil, err
	}
	state.netPrice, err = calcNetPrice(b)
	if err != nil {
		return nil, err
	}
	return &Block{
		BlockData: &BlockData{
			BlockHeader: &BlockHeader{
				parentHash: b.Hash(),
				chainID:    b.chainID,
				supply:     b.supply.DeepCopy(),
				reward:     util.NewUint128(),
				cpuPrice:   util.NewUint128(),
				cpuUsage:   0,
				netPrice:   util.NewUint128(),
				netUsage:   0,
			},
			transactions: make([]*Transaction, 0),
			height:       b.height + 1,
		},
		storage: b.storage,
		state:   state,
		sealed:  false,
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
	b.cpuPrice = b.state.cpuPrice
	b.cpuUsage = b.state.cpuUsage
	b.netPrice = b.state.netPrice
	b.netUsage = b.state.netUsage

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

	blockHash, err := HashBlockData(b.BlockData)
	if err != nil {
		return err
	}

	b.hash = blockHash
	b.sealed = true
	return nil
}

// HashBlockData returns hash of block
func HashBlockData(bd *BlockData) ([]byte, error) {
	rewardBytes, err := bd.Reward().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	supplyBytes, err := bd.Supply().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	cpuPriceBytes, err := bd.CPUPrice().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	netPriceBytes, err := bd.NetPrice().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	txHash := make([][]byte, len(bd.transactions))
	for _, tx := range bd.transactions {
		txHash = append(txHash, tx.Hash())
	}

	blockHashTarget := &corepb.BlockHashTarget{
		ParentHash:   bd.ParentHash(),
		Coinbase:     bd.Coinbase().Bytes(),
		AccStateRoot: bd.AccStateRoot(),
		TxStateRoot:  bd.TxStateRoot(),
		DposRoot:     bd.DposRoot(),
		Timestamp:    bd.Timestamp(),
		ChainId:      bd.ChainID(),
		Reward:       rewardBytes,
		Supply:       supplyBytes,
		CpuPrice:     cpuPriceBytes,
		CpuUsage:     bd.cpuUsage,
		NetPrice:     netPriceBytes,
		NetUsage:     bd.netUsage,
		TxHash:       txHash,
	}
	blockHashTargetBytes, err := proto.Marshal(blockHashTarget)
	if err != nil {
		return nil, err
	}

	return hash.GenHash(algorithm.SHA3256, blockHashTargetBytes)
}

// ExecuteTransaction on given block state
func (b *Block) ExecuteTransaction(transaction *Transaction, txMap TxFactory) (*Receipt, error) {
	// Executing process consists of two major parts
	// Part 1 : Verify transaction and not affect state trie
	// Part 2 : Execute transaction and affect state trie(store)

	// Part 1 : Verify transaction and not affect state trie

	// STEP 1. Check nonce
	if err := b.state.checkNonce(transaction); err != nil {
		return nil, err
	}

	// STEP 2. Check tx type
	newTxFunc, ok := txMap[transaction.TxType()]
	if !ok {
		return nil, ErrInvalidTransactionType
	}

	// STEP 3. Check tx components and set cpu, net usage on receipt
	tx, err := newTxFunc(transaction)
	if err != nil {
		return nil, err
	}
	cpuUsage, netUsage := tx.Bandwidth()
	cpuPoints, err := b.state.cpuPrice.Mul(util.NewUint128FromUint(cpuUsage))
	if err != nil {
		return nil, err
	}
	netPoints, err := b.state.netPrice.Mul(util.NewUint128FromUint(netUsage))
	if err != nil {
		return nil, err
	}

	points, err := cpuPoints.Add(netPoints)
	if err != nil {
		return nil, err
	}

	// STEP 4. Check bandwidth (Exceeding block's max cpu/net bandwidth)
	if err := b.state.checkBandwidthLimit(cpuUsage, netUsage); err != nil {
		return nil, err
	}

	payer, err := b.state.GetAccount(transaction.payer)
	if err != nil {
		return nil, err
	}

	// Update payer's points
	if err := payer.UpdatePoints(b.timestamp); err != nil {
		return nil, err
	}

	// STEP 5. Check payer's bandwidth
	if err := payer.checkAccountPoints(transaction, points); err != nil {
		return nil, err
	}

	// Part 2 : Execute transaction and affect state trie(store)
	// Even if transaction fails, still consume account's bandwidth

	// Update payer's bandwidth and transaction.from's unstaking status before execute transaction
	if err := b.State().PutAccount(payer); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to regenerate bandwidth.")
		return nil, err
	}
	err = b.updateUnstaking(transaction.From())
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to update staking.")
		return nil, err
	}

	receipt := &Receipt{
		executed:  false,
		timestamp: b.timestamp,
		height:    b.height,
		cpuUsage:  cpuUsage,
		netUsage:  netUsage,
		points:    points,
		error:     nil,
	}
	// Case 1. Already executed transaction payload & Execute Error (Non-system error)
	err = tx.Execute(b)
	if err != nil {
		receipt.error = []byte(err.Error())
		return receipt, err
	}
	receipt.executed = true
	return receipt, nil
}

func (b *Block) updateUnstaking(addr common.Address) error {
	acc, err := b.State().GetAccount(addr)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get account.")
		return err
	}

	if err := acc.UpdateUnstaking(b.timestamp); err != nil {
		return err
	}

	return b.State().PutAccount(acc)
}

func (b *Block) consumePoints(transaction *Transaction) error {
	var err error

	acc, err := b.State().GetAccount(transaction.payer)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get account.")
		return err
	}

	if err := acc.UpdatePoints(b.timestamp); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to update account's points")
		return err
	}

	acc.Points, err = acc.Points.Sub(transaction.receipt.points)
	if err == util.ErrUint128Underflow {
		logging.Console().WithFields(logrus.Fields{
			"tx_points":  transaction.receipt.points,
			"acc_points": acc.Points,
			"payer":      transaction.payer.Hex(),
			"err":        err,
		}).Warn("Points limit exceeded.")
		return ErrPointNotEnough
	}
	if err != nil {
		return err
	}

	err = b.State().PutAccount(acc)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to put account.")
		return err
	}
	return nil
}

// currentPoints calculates updated points based on current time.
func currentPoints(payer *Account, curTs int64) (*util.Uint128, error) {
	staking := payer.Staking
	used, err := payer.Staking.Sub(payer.Points)
	if err != nil {
		return nil, err
	}
	lastTs := payer.LastPointsTs
	elapsed := curTs - lastTs
	if time.Duration(elapsed)*time.Second >= PointsRegenerateDuration {
		return staking.DeepCopy(), nil
	}

	if elapsed < 0 {
		return nil, ErrInvalidTimestamp
	}

	if elapsed == 0 {
		return payer.Points.DeepCopy(), nil
	}

	// Points means remain points. So 0 means no used
	// regeneratedPoints = staking * elapsedTime / PointsRegenerateDuration
	// currentPoints = prevPoints + regeneratedPoints
	mul := util.NewUint128FromUint(uint64(elapsed))
	div := util.NewUint128FromUint(uint64(PointsRegenerateDuration / time.Second))
	v1, err := staking.Mul(mul)
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

	if regen.Cmp(used) >= 0 {
		return staking.DeepCopy(), nil
	}
	cur, err := payer.Points.Add(regen)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to add uint128.")
		return nil, err
	}
	return cur, nil
}

// VerifyExecution executes txs in block and verify root hashes using block header
func (b *Block) VerifyExecution(parent *Block, consensus Consensus, txMap TxFactory) error {
	err := b.BeginBatch()
	if err != nil {
		return err
	}

	if err := b.SetMintDynastyState(parent, consensus); err != nil {
		if err := b.RollBack(); err != nil {
			return err
		}
		return err
	}

	if err := consensus.VerifyProposer(b); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"blockData": b.BlockData,
			"parent":    parent,
		}).Warn("Failed to verifyProposer")
		return err
	}
	err = b.Commit()
	if err != nil {
		return err
	}
	if err := b.ExecuteAll(txMap); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to execute block transactions.")
		return err
	}

	err = b.BeginBatch()
	if err != nil {
		return err
	}
	if err := b.PayReward(b.coinbase, b.State().Supply()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to pay block reward.")
		return err
	}
	err = b.Commit()
	if err != nil {
		return err
	}

	if err := b.VerifyState(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to verify block state.")
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
	err := b.BeginBatch()
	if err != nil {
		return err
	}

	receipt, err := b.ExecuteTransaction(tx, txMap)
	if receipt == nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
		}).Warn("No Receipt from transaction execution")
		return err
	}

	if err != nil {
		if err := b.RollBack(); err != nil {
			return err
		}
	} else {
		if err := b.Commit(); err != nil {
			return err
		}
	}

	if !receipt.Equal(tx.receipt) {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
			"receipt":     receipt,
		}).Warn("transaction receipt is wrong")
		return ErrWrongReceipt
	}

	err = b.BeginBatch()
	if err != nil {
		return err
	}

	if err := b.AcceptTransaction(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
		}).Warn("Failed to accept a transaction.")
		return err
	}
	err = b.Commit()
	if err != nil {
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

// AcceptTransaction consume bandwidth and adds tx in block state
func (b *Block) AcceptTransaction(transaction *Transaction) error {
	if transaction.receipt == nil {
		return ErrNoTransactionReceipt
	}

	if err := b.consumePoints(transaction); err != nil {
		return err
	}

	b.state.cpuUsage += transaction.receipt.cpuUsage
	b.state.netUsage += transaction.receipt.netUsage

	if err := b.state.accState.incrementNonce(transaction.from); err != nil {
		return err
	}

	if err := b.state.txState.Put(transaction); err != nil {
		return err
	}

	return nil
}

// AppendTransaction append transaction to block data (only use on making block)
func (b *Block) AppendTransaction(transaction *Transaction) {
	b.BlockData.transactions = append(b.BlockData.transactions, transaction)
}

// VerifyState verifies block states comparing with root hashes in header
func (b *Block) VerifyState() error {
	if b.state.CPUPrice().Cmp(b.CPUPrice()) != 0 {
		logging.Console().WithFields(logrus.Fields{
			"state":  b.state.CPUPrice(),
			"header": b.CPUPrice(),
		}).Warn("Failed to verify CPU price.")
		return ErrInvalidCPUPrice
	}
	if b.state.NetPrice().Cmp(b.NetPrice()) != 0 {
		logging.Console().WithFields(logrus.Fields{
			"state":  b.state.NetPrice(),
			"header": b.NetPrice(),
		}).Warn("Failed to verify Net price.")
		return ErrInvalidNetPrice
	}

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
func (b *Block) SetMintDynastyState(parent *Block, consensus Consensus) error {
	mintDynasty, err := consensus.MakeMintDynasty(b.timestamp, parent)
	if err != nil {
		return err
	}
	return b.state.dposState.SetDynasty(mintDynasty)
}

// Prepare prepare block state
func (b *Block) Prepare() error {
	return b.state.prepare()
}

// BeginBatch makes block state update possible
func (b *Block) BeginBatch() error {
	return b.state.beginBatch()
}

// RollBack rolls back block state batch updates
func (b *Block) RollBack() error {
	return b.state.rollBack()
}

// Commit commit changes of block state
func (b *Block) Commit() error {
	return b.state.commit()
}

// Flush saves batch updates to storage
func (b *Block) Flush() error {
	return b.state.flush()
}

// GetBlockData returns data part of block
func (b *Block) GetBlockData() *BlockData {
	return b.BlockData
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
	reward, err := parentSupply.MulWithRat(InflationRate)
	if err != nil {
		return nil, err
	}
	roundDownDecimal := util.NewUint128FromUint(InflationRoundDown)
	reward, err = reward.Div(roundDownDecimal)
	if err != nil {
		return nil, err
	}
	reward, err = reward.Mul(roundDownDecimal)
	if err != nil {
		return nil, err
	}
	return reward, nil
}

// calculate cpu price
func calcCPUPrice(parent *Block) (*util.Uint128, error) {
	return calcBandwidthPrice(&calcBandwidthPriceArg{
		thresholdRatioNum:   ThresholdRatioNum,
		thresholdRatioDenom: ThresholdRatioDenom,
		increaseRate:        BandwidthIncreaseRate,
		decreaseRate:        BandwidthDecreaseRate,
		discountRatio:       MinimumDiscountRatio,
		limit:               CPULimit,
		usage:               parent.cpuUsage,
		supply:              parent.supply,
		previousPrice:       parent.cpuPrice,
	})
}

// calculate net price
func calcNetPrice(parent *Block) (*util.Uint128, error) {
	return calcBandwidthPrice(&calcBandwidthPriceArg{
		thresholdRatioNum:   ThresholdRatioNum,
		thresholdRatioDenom: ThresholdRatioDenom,
		increaseRate:        BandwidthIncreaseRate,
		decreaseRate:        BandwidthDecreaseRate,
		discountRatio:       MinimumDiscountRatio,
		limit:               NetLimit,
		usage:               parent.netUsage,
		supply:              parent.supply,
		previousPrice:       parent.netPrice,
	})
}

type calcBandwidthPriceArg struct {
	increaseRate, decreaseRate, discountRatio            *big.Rat
	thresholdRatioNum, thresholdRatioDenom, limit, usage uint64
	supply, previousPrice                                *util.Uint128
}

func calcBandwidthPrice(arg *calcBandwidthPriceArg) (*util.Uint128, error) {
	// thresholdBandwidth : Total MED amount which can be used for CPU / NET per block
	thresholdBandwidth := arg.limit * arg.thresholdRatioNum / arg.thresholdRatioDenom

	if arg.usage <= thresholdBandwidth {
		minPrice, err := arg.supply.Div(util.NewUint128FromUint(NumberOfBlocksInSingleTimeWindow))
		if err != nil {
			return nil, err
		}
		minPrice, err = minPrice.Div(util.NewUint128FromUint(arg.limit))
		if err != nil {
			return nil, err
		}
		minPrice, err = minPrice.MulWithRat(arg.discountRatio)
		if err != nil {
			return nil, err
		}

		newPrice, err := arg.previousPrice.MulWithRat(arg.decreaseRate)
		if err != nil {
			return nil, err
		}
		if minPrice.Cmp(newPrice) > 0 {
			return minPrice, nil
		}
		return newPrice, nil
	}

	return arg.previousPrice.MulWithRat(arg.increaseRate)
}
