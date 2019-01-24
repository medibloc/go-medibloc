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
	dState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
)

// Transaction's string representation.
const (
	TxTyGenesis = "genesis"
)

// Transaction's message types.
const (
	MessageTypeNewTx = "newtx"
)

// Block's message types.
const (
	MessageTypeNewBlock      = "newblock"
	MessageTypeRequestBlock  = "rqstblock"
	MessageTypeResponseBlock = "respblock"
)

//InflationRate is rate for reward
var InflationRate = big.NewRat(464, 100000000000) // 4.64e-09
//InflationRoundDown is constant for round down reward
const InflationRoundDown = 10000000000 //1e10

// Points limit per block
const (
	NumberOfBlocksInSingleTimeWindow = 201600  // 7 * 86400 / 3 (time window: 7days, block interval: 3 sec)
	CPULimit                         = 3000000 // 1000 TPS (block interval: 3 sec, transfer tx: 1000 )
	NetLimit                         = 3000000 // 3MB
)

// Points Price related defaults
var (
	MinimumDiscountRatio  = big.NewRat(1, 100)
	BandwidthIncreaseRate = big.NewRat(105, 100)
	BandwidthDecreaseRate = big.NewRat(95, 100)
)

// Points Price related defaults
const (
	ThresholdRatioNum   = 5
	ThresholdRatioDenom = 10
)

// Error types of core package.
var (
	ErrNotFound                        = storage.ErrKeyNotFound
	ErrBeginAgainInBatch               = errors.New("cannot begin with a batch task unfinished")
	ErrCannotCloneOnBatching           = errors.New("cannot clone on batching")
	ErrNotBatching                     = errors.New("not batching")
	ErrFailedValidateHeightAndHeight   = errors.New("failed to verify height and timestamp by lib")
	ErrCannotRemoveBlockOnCanonical    = errors.New("cannot remove block on canonical chain")
	ErrDuplicatedBlock                 = errors.New("duplicated block")
	ErrDuplicatedTransaction           = errors.New("duplicated transaction")
	ErrGenesisNotMatch                 = errors.New("genesis block does not match")
	ErrInvalidProtoToBlock             = errors.New("protobuf message cannot be converted into Block")
	ErrInvalidProtoToBlockHeader       = errors.New("protobuf message cannot be converted into BlockHeader")
	ErrInvalidBlockChainID             = errors.New("invalid block chainID")
	ErrInvalidBlockToProto             = errors.New("block cannot be converted into proto")
	ErrInvalidBlockHash                = errors.New("invalid block hash")
	ErrInvalidTimestamp                = errors.New("child block's timestamp is smaller than parent block's")
	ErrBlockAlreadySealed              = errors.New("cannot seal an already sealed block")
	ErrNilArgument                     = errors.New("argument(s) is nil")
	ErrLargeTransactionNonce           = errors.New("transaction nonce is larger than expected")
	ErrSmallTransactionNonce           = errors.New("transaction nonce is smaller than expected")
	ErrMissingParentBlock              = errors.New("cannot find the block's parent block in storage")
	ErrBlockNotExist                   = errors.New("block not exist")
	ErrBlockNotSealed                  = errors.New("block should be sealed first to be signed")
	ErrInvalidBlockHeight              = errors.New("block height should be one block higher than the parent")
	ErrInvalidBlockReward              = errors.New("invalid reward")
	ErrInvalidBlockSupply              = errors.New("invalid supply")
	ErrInvalidBlockAccountsRoot        = errors.New("invalid account state root hash")
	ErrInvalidBlockTxsRoot             = errors.New("invalid transactions state root hash")
	ErrInvalidBlockDposRoot            = errors.New("invalid block dpos root hash")
	ErrTooOldTransaction               = errors.New("transaction timestamp is too old")
	ErrBlockSignatureNotExist          = errors.New("block signature does not exist in the blockheader")
	ErrFailedToDirectPush              = errors.New("cannot direct push to chain")
	ErrExceedBlockMaxCPUUsage          = errors.New("transaction exceeds block's max cpu usage")
	ErrExceedBlockMaxNetUsage          = errors.New("transaction exceeds block's max net usage")
	ErrNoTransactionReceipt            = errors.New("failed to load transaction receipt")
	ErrWrongReceipt                    = errors.New("transaction receipt is wrong in block data")
	ErrInvalidCPUPrice                 = errors.New("invalid cpu price")
	ErrInvalidNetPrice                 = errors.New("invalid Net price")
	ErrInvalidCPUUsage                 = errors.New("block uses too much cpu bandwidth")
	ErrInvalidNetUsage                 = errors.New("block ueses too much net bandwidth")
	ErrWrongCPUUsage                   = errors.New("block cpu usage is not matched with sum of tx cpu usage")
	ErrWrongNetUsage                   = errors.New("block net usage is not matched with sum of tx net usage")
	ErrInvalidAlias                    = errors.New("invalid alias")
	ErrInvalidHash                     = errors.New("invalid hash")
	ErrFailedToReplacePendingTx        = errors.New("cannot replace pending transaction in 10 minute")
	ErrBlockExecutionTimeout           = errors.New("block is not executed on time")
	ErrAlreadyOnTheChain               = errors.New("block is already on the chain")
	ErrCannotFindParentBlockOnTheChain = errors.New("cannot find parent block on the chain")
	ErrForkedBeforeLIB                 = errors.New("block is forked before LIB")
	ErrInvalidBlock                    = errors.New("invalid block")
	ErrSameDynasty                     = errors.New("new block is in same dynasty with parent block")
	ErrTxTypeInvalid                   = errors.New("invalid transaction type")
	ErrNotGenesisBlock                 = errors.New("block is not genesis")
)

// HashableBlock is an interface that can get its own or parent's hash.
type HashableBlock interface {
	Hash() []byte
	ParentHash() []byte
}

// Consensus is an interface of consensus model
type Consensus interface {
	NewConsensusState(dposRootBytes []byte, stor storage.Storage) (*dState.State, error)

	DynastySize() int
	MakeMintDynasty(ts int64, parentState *BlockState) ([]common.Address, error)

	VerifyHeightAndTimestamp(lib, bd *BlockData) error
	MissingBlocks(lib, bd *BlockData) uint64
	VerifyInterval(bd *BlockData, parent *Block) error
	VerifyProposer(b *Block) error
	FindLIB(bc *BlockChain) (newLIB *Block)
	FindMintProposer(ts int64, parent *Block) (common.Address, error)
}

// SyncService interface for sync
type SyncService interface {
	Download(bd *BlockData) error
	IsDownloadActivated() bool
}

type Canonical interface {
	TailBlock() *Block
}

// ExecutableTx interface for execute transaction on state
type ExecutableTx interface {
	Execute(b *Block) error
	Bandwidth() *common.Bandwidth
	PointModifier(points *util.Uint128) (modifiedPoints *util.Uint128, err error)
	RecoverFrom() (common.Address, error)
}
