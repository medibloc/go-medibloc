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
	"errors"
	"math/big"

	"github.com/medibloc/go-medibloc/common"
	corestate "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Block represents block with actual state tries
type Block struct {
	*BlockData
	state  *BlockState
	sealed bool
}

func (b *Block) FromBlockData(bd *BlockData, consensus Consensus, storage storage.Storage) error {
	var err error
	b.BlockData = bd
	b.state, err = NewBlockState(bd, consensus, storage)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": bd,
			"err":   err,
		}).Error("Failed to create new block state.")
		return err
	}
	b.sealed = true
	return nil
}

// InitChild return initial child block for verifying or making block
func (b *Block) InitChild(coinbase common.Address) (*Block, error) {
	bs, err := b.state.Clone()
	if err != nil {
		return nil, err
	}
	bs.cpuUsage = 0
	bs.netUsage = 0

	bs.cpuPrice, err = calcCPUPrice(b)
	if err != nil {
		return nil, err
	}
	bs.netPrice, err = calcNetPrice(b)
	if err != nil {
		return nil, err
	}
	return &Block{
		BlockData: &BlockData{
			BlockHeader: &BlockHeader{
				hash:         nil,
				parentHash:   b.Hash(),
				accStateRoot: nil,
				txStateRoot:  nil,
				dposRoot:     nil,
				coinbase:     coinbase,
				reward:       util.NewUint128(),
				supply:       b.supply.DeepCopy(),
				timestamp:    0,
				chainID:      b.chainID,
				sign:         nil,
				cpuPrice:     util.NewUint128(),
				cpuUsage:     0,
				netPrice:     util.NewUint128(),
				netUsage:     0,
			},
			transactions: make([]*Transaction, 0),
			height:       b.height + 1,
		},
		state:  bs,
		sealed: false,
	}, nil
}

// CreateChildWithBlockData returns child block by executing block data on parent block.
func (b *Block) CreateChildWithBlockData(bd *BlockData, consensus Consensus) (child *Block, err error) {
	// Prepare Execution
	child, err = b.InitChild(bd.coinbase)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to make child block for execution on parent block")
		return nil, err
	}
	child.BlockData = bd
	child.State().SetTimestamp(bd.timestamp)
	// TODO call block.Timestamp() instead

	err = child.Prepare()
	if err != nil {
		return nil, err
	}

	if err := child.verifyExecution(b, consensus); err != nil {
		return nil, err
	}
	err = child.Flush()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to flush state")
		return nil, err
	}
	return child, nil
}

// State returns block state
func (b *Block) State() *BlockState {
	return b.state
}

// Sealed returns sealed
func (b *Block) Sealed() bool {
	return b.sealed
}

// Seal writes state root hashes and block hash in block header
func (b *Block) Seal() error {
	var err error
	if b.sealed {
		return ErrBlockAlreadySealed
	}

	b.timestamp = b.state.timestamp
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

	blockHash, err := b.CalcHash()
	if err != nil {
		return err
	}

	b.hash = blockHash
	b.sealed = true
	return nil
}

var (
	ErrAtomicError = errors.New("failed to process atomic operation")
)

// ExecuteTransaction on given block state
func (b *Block) ExecuteTransaction(tx *Transaction) (*Receipt, error) {
	// Verify that transaction is executable
	exeTx, err := TxConv(tx)
	if err != nil {
		return nil, err
	}

	if err := b.checkNonce(tx); err != nil {
		return nil, err
	}

	if err := b.checkBandwidth(exeTx); err != nil {
		return nil, err
	}

	usage, err := b.calcPointUsage(exeTx)
	if err != nil {
		return nil, err
	}

	if err := b.checkAvailablePoint(tx, exeTx, usage); err != nil {
		return nil, err
	}

	execErr, err := b.Atomic(exeTx.Execute)
	if err != nil {
		return nil, ErrAtomicError
	}
	if execErr != nil {
		return b.makeErrorReceipt(exeTx.Bandwidth(), usage, execErr), nil
	}
	return b.makeSuccessReceipt(exeTx.Bandwidth(), usage), nil
}

func (b *Block) receiptTemplate(bw *common.Bandwidth, point *util.Uint128) *Receipt {
	receipt := new(Receipt)
	receipt.SetTimestamp(b.state.timestamp)
	receipt.SetHeight(b.Height())
	receipt.SetCPUUsage(bw.CPUUsage())
	receipt.SetNetUsage(bw.NetUsage())
	receipt.SetPoints(point)
	receipt.SetExecuted(false)
	receipt.SetError(nil)
	return receipt
}

func (b *Block) makeErrorReceipt(bw *common.Bandwidth, point *util.Uint128, err error) *Receipt {
	receipt := b.receiptTemplate(bw, point)
	receipt.SetError([]byte(err.Error()))
	return receipt
}

func (b *Block) makeSuccessReceipt(bw *common.Bandwidth, point *util.Uint128) *Receipt {
	receipt := b.receiptTemplate(bw, point)
	receipt.SetExecuted(true)
	return receipt
}

// TODO move to types.go
var ErrNonceNotExecutable = errors.New("transaction nonce not executable")

func (b *Block) checkNonce(tx *Transaction) error {
	from, err := b.state.GetAccount(tx.From())
	if err != nil {
		return err
	}
	if tx.Nonce() != from.Nonce+1 {
		return ErrNonceNotExecutable
	}
	return nil
}

func (b *Block) checkBandwidth(exeTx ExecutableTx) error {
	if err := b.state.checkBandwidthLimit(exeTx.Bandwidth()); err != nil {
		return err
	}
	return nil
}

func (b *Block) checkAvailablePoint(tx *Transaction, exeTx ExecutableTx, usage *util.Uint128) error {
	if tx.HasPayer() {
		payer, err := b.state.GetAccount(tx.Payer())
		if err != nil {
			return err
		}
		return b.checkPayerAccountPoint(payer, usage)
	}

	from, err := b.state.GetAccount(tx.From())
	if err != nil {
		return err
	}
	return b.checkFromAccountPoint(from, exeTx, usage)
}

func (b *Block) checkPayerAccountPoint(payer *corestate.Account, usage *util.Uint128) error {
	if payer.Points.Cmp(usage) < 0 {
		return ErrPointNotEnough
	}
	return nil
}

func (b *Block) checkFromAccountPoint(from *corestate.Account, exeTx ExecutableTx, usage *util.Uint128) (err error) {
	avail := from.Points
	neg, abs := exeTx.PointChange()
	if neg {
		avail, err = avail.Sub(abs)
	} else {
		avail, err = avail.Add(abs)
	}
	if err != nil && err != util.ErrUint128Underflow {
		return err
	}
	if err == util.ErrUint128Underflow {
		avail = util.Uint128Zero()
	}
	if avail.Cmp(usage) < 0 {
		return ErrPointNotEnough
	}
	return nil
}

func (b *Block) calcPointUsage(exeTx ExecutableTx) (*util.Uint128, error) {
	return exeTx.Bandwidth().CalcPoints(b.state.Price())
}

// verifyExecution executes txs in block and verify root hashes using block header
func (b *Block) verifyExecution(parent *Block, consensus Consensus) error {
	if err := b.SetMintDynasty(parent, consensus); err != nil {
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

	if err := b.executeAll(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to execute block transactions.")
		return err
	}

	if err := b.PayReward(parent.Supply()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to pay block reward.")
		return err
	}

	if err := b.verifyState(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": b,
		}).Error("Failed to verify block state.")
		return err
	}

	return nil
}

// executeAll executes all txs in block
func (b *Block) executeAll() error {
	for _, transaction := range b.transactions {
		err := b.execute(transaction)
		if err != nil {
			return err
		}
	}

	return nil
}

// execute executes a transaction.
func (b *Block) execute(tx *Transaction) error {
	receipt, err := b.ExecuteTransaction(tx)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
		}).Debug("Failed to execute transaction.")
		return err
	}

	if !receipt.Equal(tx.Receipt()) {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
			"receipt":     receipt,
		}).Warn("transaction receipt is wrong")
		return ErrWrongReceipt
	}

	if err := b.AcceptTransaction(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": tx,
			"block":       b,
		}).Warn("Failed to accept a transaction.")
		return err
	}
	return nil
}

func (b *Block) SetMintDynasty(parent *Block, consensus Consensus) error {
	mintDynasty, err := consensus.MakeMintDynasty(b.state.timestamp, parent.State())
	if err != nil && err == ErrSameDynasty {
		return nil
	}
	if err != nil {
		return err
	}

	execErr, err := b.Atomic(func(block *Block) error {
		return block.state.DposState().SetDynasty(mintDynasty)
	})
	if err != nil {
		return err
	}
	if execErr != nil {
		return execErr
	}
	return nil
}

func (b *Block) AcceptTransaction(tx *Transaction) error {
	if tx.Receipt() == nil {
		return ErrNoTransactionReceipt
	}

	from, err := b.state.GetAccount(tx.From())
	if err != nil {
		return err
	}
	from.Nonce++

	var payer *corestate.Account
	if tx.HasPayer() {
		payer, err = b.state.GetAccount(tx.Payer())
		if err != nil {
			return err
		}
	} else {
		payer = from
	}

	payer.Points, err = payer.Points.Sub(tx.Receipt().Points())
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tx":    tx,
			"payer": payer,
			"err":   err,
		}).Error("Failed to subtract point.")
		return err
	}

	execErr, err := b.Atomic(func(block *Block) error {
		if err := block.state.PutAccount(from); err != nil {
			return err
		}
		if tx.HasPayer() {
			if err := block.state.PutAccount(payer); err != nil {
				return err
			}
		}
		return block.state.PutTx(tx)
	})
	if err != nil {
		return err
	}
	if execErr != nil {
		return err
	}

	b.state.cpuUsage += tx.Receipt().CPUUsage()
	b.state.netUsage += tx.Receipt().NetUsage()

	return nil
}

var ErrCoinbaseNotSet = errors.New("coinbase is not set")

// PayReward add reward to coinbase and update reward and supply
func (b *Block) PayReward(parentSupply *util.Uint128) error {
	if b.coinbase.Equals(common.Address{}) {
		return ErrCoinbaseNotSet
	}

	reward, err := calcMintReward(parentSupply)
	if err != nil {
		return err
	}
	supply, err := parentSupply.Add(reward)
	if err != nil {
		return err
	}

	acc, err := b.state.GetAccount(b.coinbase)
	if err != nil {
		return err
	}
	acc.Balance, err = acc.Balance.Add(reward)
	if err != nil {
		return err
	}

	execErr, err := b.Atomic(func(block *Block) error {
		return block.state.PutAccount(acc)
	})
	if err != nil {
		return err
	}
	if execErr != nil {
		return execErr
	}

	b.state.SetReward(reward)
	b.state.SetSupply(supply)

	return nil
}

// calcMintReward returns calculated block produce reward
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

// verifyState verifies block states comparing with root hashes in header
func (b *Block) verifyState() error {
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

func (b *Block) Atomic(batch func(block *Block) error) (execErr error, err error) {
	if err := b.BeginBatch(); err != nil {
		return nil, ErrAtomicError
	}
	if execErr := batch(b); execErr != nil {
		if err := b.RollBack(); err != nil {
			return nil, ErrAtomicError
		}
		return execErr, nil
	}
	if err := b.Commit(); err != nil {
		return nil, ErrAtomicError
	}
	return nil, nil
}

// GetBlockData returns data part of block
func (b *Block) GetBlockData() *BlockData {
	return b.BlockData
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
