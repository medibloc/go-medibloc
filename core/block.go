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

	cpuRef   *util.Uint128
	cpuUsage *util.Uint128
	netRef   *util.Uint128
	netUsage *util.Uint128
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

	cpuRef, err := b.cpuRef.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	netRef, err := b.netRef.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	cpuUsage, err := b.cpuUsage.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	netUsage, err := b.netUsage.ToFixedSizeByteSlice()
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
		CpuRef:       cpuRef,
		CpuUsage:     cpuUsage,
		NetRef:       netRef,
		NetUsage:     netUsage,
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

		cpuRef, err := util.NewUint128FromFixedSizeByteSlice(msg.CpuRef)
		if err != nil {
			return err
		}

		cpuUsage, err := util.NewUint128FromFixedSizeByteSlice(msg.CpuUsage)
		if err != nil {
			return err
		}

		netRef, err := util.NewUint128FromFixedSizeByteSlice(msg.NetRef)
		if err != nil {
			return err
		}

		netUsage, err := util.NewUint128FromFixedSizeByteSlice(msg.NetUsage)
		if err != nil {
			return err
		}

		b.cpuRef = cpuRef
		b.cpuUsage = cpuUsage
		b.netRef = netRef
		b.netUsage = netUsage
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

//CPURef returns cpuRef
func (b *BlockHeader) CPURef() *util.Uint128 {
	return b.cpuRef
}

//SetCPURef sets cpuRef
func (b *BlockHeader) SetCPURef(cpuRef *util.Uint128) {
	b.cpuRef = cpuRef
}

//NetRef returns netRef
func (b *BlockHeader) NetRef() *util.Uint128 {
	return b.netRef
}

//SetNetRef sets netRef
func (b *BlockHeader) SetNetRef(netRef *util.Uint128) {
	b.netRef = netRef
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

// CPUUsage returns cpuUsage
func (b *BlockHeader) CPUUsage() *util.Uint128 {
	return b.cpuUsage
}

// NetUsage returns netUsage
func (b *BlockHeader) NetUsage() *util.Uint128 {
	return b.netUsage
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
	return fmt.Sprintf("<height:%v, hash:%v, parent_hash:%v, coinbase:%v, reward:%v, supply:%v, timestamp:%v, "+
		"proposer:%v, cpuBandwidth:%v, netBandwidth:%v>",
		bd.Height(),
		byteutils.Bytes2Hex(bd.Hash()),
		byteutils.Bytes2Hex(bd.ParentHash()),
		byteutils.Bytes2Hex(bd.Coinbase().Bytes()),
		bd.Reward().String(),
		bd.Supply().String(),
		bd.Timestamp(),
		proposer.Hex(),
		bd.CPUUsage(),
		bd.NetUsage(),
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
func (bd *BlockData) ExecuteOnParentBlock(parent *Block, txMap TxFactory) (*Block, error) {
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

	if err := block.VerifyExecution(parent, txMap); err != nil {
		block.Storage().DisableBatch()
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
		consensus: consensus,
	}
	if block.state, err = newStates(block.consensus, storage); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to create new block state.")
		return nil, err
	}

	block.state.reward = bd.Reward()
	block.state.supply = bd.Supply()

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

//Child return initial child block for verifying or making block
func (b *Block) Child() (*Block, error) {
	state, err := b.state.Clone()
	if err != nil {
		return nil, err
	}
	cpuRef, err := calcRefCPU(b)
	if err != nil {
		return nil, err
	}
	netRef, err := calcRefNet(b)
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
				cpuRef:     cpuRef,
				cpuUsage:   util.NewUint128(),
				netRef:     netRef,
				netUsage:   util.NewUint128(),
			},
			transactions: make([]*Transaction, 0),
			height:       b.height + 1,
		},
		storage:   b.storage,
		state:     state,
		consensus: b.consensus,
		sealed:    false,
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

	hash, err := HashBlockData(b.BlockData)
	if err != nil {
		return err
	}

	b.hash = hash
	b.sealed = true
	return nil
}

// HashBlockData returns hash of block
func HashBlockData(bd *BlockData) ([]byte, error) {
	hasher := sha3.New256()

	hasher.Write(bd.ParentHash())
	hasher.Write(bd.Coinbase().Bytes())
	hasher.Write(bd.AccStateRoot())
	hasher.Write(bd.TxStateRoot())
	hasher.Write(bd.DposRoot())
	hasher.Write(byteutils.FromInt64(bd.Timestamp()))
	hasher.Write(byteutils.FromUint32(bd.ChainID()))

	rewardBytes, err := bd.Reward().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	supplyBytes, err := bd.Supply().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	cpuUsageBytes, err := bd.CPUUsage().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	netUsageBytes, err := bd.NetUsage().ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	hasher.Write(rewardBytes)
	hasher.Write(supplyBytes)
	// hasher.Write(bd.Alg())
	hasher.Write(cpuUsageBytes)
	hasher.Write(cpuUsageBytes)
	hasher.Write(netUsageBytes)
	hasher.Write(netUsageBytes)

	for _, tx := range bd.transactions {
		hasher.Write(tx.Hash())
	}

	return hasher.Sum(nil), nil
}

// VerifyTransaction verifies transaction before execute
// This process doesn't change account's nonce and bandwidth
func (b *Block) VerifyTransaction(transaction *Transaction, txMap TxFactory) error {
	// Case 1. Unmatched Nonce
	if err := b.state.checkNonce(transaction); err != nil {
		return err
	}

	// Case 2, 3, 4 are checked in the newTxFunc method
	// Case 2. Invalid TX type
	// Case 3. Invalid Components Error (Payload)
	// Case 4. Too large TX size
	newTxFunc, ok := txMap[transaction.TxType()]
	if !ok {
		return ErrInvalidTransactionType
	}
	tx, err := newTxFunc(transaction)
	if err != nil {
		return err
	}

	// Case 5. Lack of balance
	// TODO @ggomma consider every transaction case
	acc, err := b.state.GetAccount(transaction.From())
	if err != nil {
		return err
	}
	if acc.Balance.Cmp(transaction.Value()) < 0 {
		return ErrBalanceNotEnough
	}

	// Case 6. Lack of bandwidth (transaction.from & payer)
	payer, err := transaction.recoverPayer()
	if err == ErrPayerSignatureNotExist {
		payer = transaction.From()
	} else if err != nil {
		return err
	}
	payerAcc, err := b.state.GetAccount(payer)
	if err != nil {
		return err
	}
	curBandwidth, err := currentBandwidth(payerAcc.Vesting, payerAcc.Bandwidth, payerAcc.LastBandwidthTs, b.Timestamp())
	if err != nil {
		return err
	}
	if transaction.TxType() == TxOpVest {
		payerAcc.Vesting, err = payerAcc.Vesting.Add(transaction.Value())
		if err != nil {
			return err
		}
	}
	avail, err := payerAcc.Vesting.Sub(curBandwidth)
	if err != nil {
		return err
	}
	cpuUsage, netUsage, err := tx.Bandwidth()
	if err != nil {
		return err
	}
	usage, err := cpuUsage.Add(netUsage)
	if err != nil {
		return err
	}
	if avail.Cmp(usage) < 0 {
		return ErrBandwidthLimitExceeded
	}

	return nil
}

// ExecuteTransaction on given block state
func (b *Block) ExecuteTransaction(transaction *Transaction, txMap TxFactory) error {
	newTxFunc, _ := txMap[transaction.TxType()]
	tx, _ := newTxFunc(transaction)

	// Case 1. Exceed block's max bandwidth - return to tx pool
	// Exceed max cpu || net
	if err := b.checkBandwidthLimit(tx); err != nil {
		return err
	}

	// Update payer's bandwidth and transaction.from's unstaking status before execute transaction
	payer, err := transaction.recoverPayer()
	if err == ErrPayerSignatureNotExist {
		payer = transaction.From()
	}
	if err := b.regenerateBandwidth(payer); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to regenerate bandwidth.")
		return err
	}

	err = b.updateUnstaking(transaction.From())
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to update staking.")
		return err
	}

	// Case 2. Already executed transaction payload
	// Case 3. Execute Error (Non-system error)
	cpuUsage, netUsage, err := tx.Bandwidth()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get bandwidth of a transaction.")
		return err
	}
	usage, err := cpuUsage.Add(netUsage)
	if err != nil {
		return err
	}

	receipt := &Receipt{
		executed: true,
		cpuUsage: cpuUsage,
		netUsage: netUsage,
		error:    nil,
	}
	transaction.SetReceipt(receipt)

	err = tx.Execute(b)
	if err != nil {
		receipt.executed = false
		receipt.error = []byte(err.Error())
		transaction.SetReceipt(receipt)
	}

	err = b.consumeBandwidth(payer, usage)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to update bandwidth.")

		receipt.executed = false
		receipt.error = []byte(err.Error())
		transaction.SetReceipt(receipt)
		return ErrExecutedErr
	}

	if receipt.error == nil {
		return nil
	}

	return ErrExecutedErr
}

func (b *Block) updateUnstaking(addr common.Address) error {
	acc, err := b.State().GetAccount(addr)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to get account.")
		return err
	}

	// Unstaking action does not exist
	if acc.LastUnstakingTs == 0 {
		return nil
	}

	// Staked coin is not returned if not enough time has been passed
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

	err = b.State().PutAccount(acc) // TODO @ggomma use temporary state
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

	// Success if (vesting - bandwidth > usage)
	avail, err := acc.Vesting.Sub(acc.Bandwidth)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to calculate available usage.")
		return err
	}
	if avail.Cmp(usage) < 0 {
		logging.Console().WithFields(logrus.Fields{
			"usage": usage,
			"avail": avail,
			"payer": addr.Hex(),
			"err":   err,
		}).Warn("Bandwidth limit exceeded.")
		return ErrBandwidthLimitExceeded
	}

	// Update bandwidth and lastBandwidthTimestamp
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

	// TODO @ggomma use temporary state
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

	// update account
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

// currentBandwidth calculates updated bandwidth based on current time.
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

	// Bandwidth means consumed bandwidth. So 0 means full bandwidth
	// regeneratedBandwidth = veting * elapsedTime / BandwidthRegenerateDuration
	// currentBandwidth = prevBandwidth - regeneratedBandwidth
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
		return err
	}

	if err := b.PayReward(b.coinbase, b.State().Supply()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to pay block reward.")
		return err
	}

	b.Commit()

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
	if err := b.ExecuteTransaction(tx, txMap); err != nil {
		if err != ErrExecutedErr {
			logging.Console().WithFields(logrus.Fields{
				"err":         err,
				"transaction": tx,
				"block":       b,
			}).Warn("Failed to execute a transaction.")
			return err
		}
	}

	if err := b.state.acceptTransaction(tx); err != nil {
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
func (b *Block) AcceptTransaction(transaction *Transaction, txMap TxFactory) error {
	if err := b.state.acceptTransaction(transaction); err != nil {
		return err
	}
	b.transactions = append(b.transactions, transaction)

	if txMap != nil {
		newTxFunc, _ := txMap[transaction.TxType()]
		tx, _ := newTxFunc(transaction)
		cpuUsage, netUsage, err := tx.Bandwidth()
		if err != nil {
			return err
		}

		b.cpuUsage, err = b.cpuUsage.Add(cpuUsage)
		if err != nil {
			return err
		}
		b.netUsage, err = b.netUsage.Add(netUsage)
		if err != nil {
			return err
		}
	}

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

func (b *Block) checkBandwidthLimit(tx ExecutableTx) error {
	// TODO @ggomma calculate cpu / net usage from transaction
	cpuUsage, netUsage, err := tx.Bandwidth()
	if err != nil {
		return err
	}

	// TODO @ggomma use tx.cpuUsage and tx.netUsage
	totalCPUUsage, err := b.cpuUsage.Add(cpuUsage)
	if err != nil {
		return err
	}
	maxCPU, err := util.NewUint128FromString(cpuLimit)
	if err != nil {
		return err
	}
	if maxCPU.Cmp(totalCPUUsage) < 0 {
		return ErrExceedBlockMaxCPUUsage
	}

	totalNetUsage, err := b.netUsage.Add(netUsage)
	if err != nil {
		return err
	}
	maxNet, err := util.NewUint128FromString(netLimit)
	if err != nil {
		return err
	}
	if maxNet.Cmp(totalNetUsage) < 0 {
		return ErrExceedBlockMaxNETUsage
	}
	return nil
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
	reward, err := parentSupply.MulWithFloat(InflationRate)
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

// calculate Reference cpu Bandwidth

func calcRefCPU(parent *Block) (*util.Uint128, error) {
	limit, err := util.NewUint128FromString(cpuLimit)
	if err != nil {
		return nil, err
	}
	return calcRefBandwidth(&calcRefBandwidthArg{
		thresholdRatio: ThresholdRatio,
		increaseRate:   BandwidthIncreaseRate,
		decreaseRate:   BandwidthDecreaseRate,
		discountRatio:  MinimumDiscountRatio,
		limit:          limit,
		usage:          parent.cpuUsage,
		supply:         parent.supply,
		previousRef:    parent.cpuRef,
	})
}

// calculate Reference net Bandwidth
func calcRefNet(parent *Block) (*util.Uint128, error) {
	limit, err := util.NewUint128FromString(netLimit)
	if err != nil {
		return nil, err
	}
	return calcRefBandwidth(&calcRefBandwidthArg{
		thresholdRatio: ThresholdRatio,
		increaseRate:   BandwidthIncreaseRate,
		decreaseRate:   BandwidthDecreaseRate,
		discountRatio:  MinimumDiscountRatio,
		limit:          limit,
		usage:          parent.netUsage,
		supply:         parent.supply,
		previousRef:    parent.netRef,
	})
}

type calcRefBandwidthArg struct {
	thresholdRatio, increaseRate, decreaseRate, discountRatio string
	limit, usage, supply, previousRef                         *util.Uint128
}

func calcRefBandwidth(arg *calcRefBandwidthArg) (*util.Uint128, error) {

	thresholdBandwidth, err := arg.limit.MulWithFloat(arg.thresholdRatio)
	if err != nil {
		return nil, err
	}

	// Set Reference Net Bandwidth
	if arg.usage.Cmp(thresholdBandwidth) < 0 {
		minRef, err := arg.supply.Div(util.NewUint128FromUint(NumberOfBlocksInSingleTimeWindow))
		if err != nil {
			return nil, err
		}
		minRef, err = arg.supply.Div(arg.limit)
		if err != nil {
			return nil, err
		}
		minRef, err = minRef.MulWithFloat(arg.discountRatio)
		if err != nil {
			return nil, err
		}

		newRef, err := arg.previousRef.MulWithFloat(arg.decreaseRate)
		if err != nil {
			return nil, err
		}
		if minRef.Cmp(newRef) > 0 {
			return minRef, nil
		}
		return newRef, nil
	}

	return arg.previousRef.MulWithFloat(arg.increaseRate)
}
