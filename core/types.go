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
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
)

// Transaction's string representation.
const (
	TxTyGenesis             = "genesis"
	TxTyGenesisVesting      = "genesis_vest"
	TxOpTransfer            = "transfer"
	TxOpAddRecord           = "add_record"
	TxOpVest                = "vest"
	TxOpWithdrawVesting     = "withdraw_vesting"
	TxOpAddCertification    = "add_certification"
	TxOpRevokeCertification = "revoke_certification"
	TxOpRegisterAlias       = "register_alias"
	TxOpDeregisterAlias     = "Deregister_alias"
)

// Transaction related defaults
const (
	TxDelayLimit                = 24 * 60 * 60
	UnstakingWaitDuration       = 7 * 24 * time.Hour
	BandwidthRegenerateDuration = 7 * 24 * time.Hour
	MaxPayloadSize              = 4096
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
var InflationRate = big.NewRat(464, 1000000000000) // 4.64e-09
//InflationRoundDown is constant for round down reward
const InflationRoundDown = 10000000000 //1e10

// Bandwidth limit per block
const (
	NumberOfBlocksInSingleTimeWindow = 201600  // 7 * 86400 / 3 (time window: 7days, block interval: 3 sec)
	CPULimit                         = 3000000 // 1000 TPS (block interval: 3 sec, transfer tx: 1000 )
	NetLimit                         = 3000000 // 3MB
)

// Bandwidth related defaults
var (
	MinimumDiscountRatio  = big.NewRat(1, 100)
	BandwidthIncreaseRate = big.NewRat(105, 100)
	BandwidthDecreaseRate = big.NewRat(95, 100)
	ThresholdRatio        = big.NewRat(5, 10)
)

// Error types of core package.
var (
	ErrNotFound                         = storage.ErrKeyNotFound
	ErrBalanceNotEnough                 = errors.New("balance is not enough")
	ErrBeginAgainInBatch                = errors.New("cannot begin with a batch task unfinished")
	ErrCannotCloneOnBatching            = errors.New("cannot clone on batching")
	ErrInvalidAmount                    = errors.New("invalid amount")
	ErrNotBatching                      = errors.New("not batching")
	ErrVestingNotEnough                 = errors.New("vesting is not enough")
	ErrBandwidthNotEnough               = errors.New("bandwidth limit exceeded")
	ErrCannotConvertTransaction         = errors.New("proto message cannot be converted into Transaction")
	ErrCannotRevertLIB                  = errors.New("cannot revert latest irreversible block")
	ErrCannotRemoveBlockOnCanonical     = errors.New("cannot remove block on canonical chain")
	ErrCannotExecuteOnParentBlock       = errors.New("cannot execute on parent block")
	ErrDuplicatedBlock                  = errors.New("duplicated block")
	ErrDuplicatedTransaction            = errors.New("duplicated transaction")
	ErrGenesisNotMatch                  = errors.New("genesis block does not match")
	ErrInvalidTransactionHash           = errors.New("invalid transaction hash")
	ErrInvalidTransactionSigner         = errors.New("transaction recover public key address not equal to from")
	ErrInvalidTransactionType           = errors.New("invalid transaction type")
	ErrInvalidProtoToBlock              = errors.New("protobuf message cannot be converted into Block")
	ErrInvalidProtoToBlockHeader        = errors.New("protobuf message cannot be converted into BlockHeader")
	ErrInvalidChainID                   = errors.New("invalid transaction chainID")
	ErrTransactionHashFailed            = errors.New("failed to hash transaction")
	ErrInvalidBlockToProto              = errors.New("block cannot be converted into proto")
	ErrInvalidBlockHash                 = errors.New("invalid block hash")
	ErrInvalidTimestamp                 = errors.New("child block's timestamp is smaller than parent block's")
	ErrBlockAlreadySealed               = errors.New("cannot seal an already sealed block")
	ErrNilArgument                      = errors.New("argument(s) is nil")
	ErrVoidTransaction                  = errors.New("nothing to do with transaction")
	ErrLargeTransactionNonce            = errors.New("transaction nonce is larger than expected")
	ErrSmallTransactionNonce            = errors.New("transaction nonce is smaller than expected")
	ErrMissingParentBlock               = errors.New("cannot find the block's parent block in storage")
	ErrBlockNotExist                    = errors.New("block not exist")
	ErrBlockNotSealed                   = errors.New("block should be sealed first to be signed")
	ErrInvalidBlockHeight               = errors.New("block height should be one block higher than the parent")
	ErrInvalidBlockReward               = errors.New("invalid reward")
	ErrInvalidBlockSupply               = errors.New("invalid supply")
	ErrInvalidBlockAccountsRoot         = errors.New("invalid account state root hash")
	ErrInvalidBlockTxsRoot              = errors.New("invalid transactions state root hash")
	ErrInvalidBlockRecordsRoot          = errors.New("invalid records state root hash")
	ErrInvalidBlockCertificationRoot    = errors.New("invalid certification state root hash")
	ErrInvalidBlockDposRoot             = errors.New("invalid block dpos root hash")
	ErrTooOldTransaction                = errors.New("transaction timestamp is too old")
	ErrFailedToUnmarshalPayload         = errors.New("cannot unmarshal tx payload")
	ErrFailedToMarshalPayload           = errors.New("cannot marshal tx payload to bytes")
	ErrCheckPayloadIntegrity            = errors.New("payload has invalid elements")
	ErrTooLargePayload                  = errors.New("too large payload")
	ErrRecordAlreadyAdded               = errors.New("record hash already added")
	ErrRecordReaderAlreadyAdded         = errors.New("record reader hash already added")
	ErrCertReceivedAlreadyAdded         = errors.New("hash of received cert already added")
	ErrCertIssuedAlreadyAdded           = errors.New("hash of issued cert already added")
	ErrCertAlreadyRevoked               = errors.New("cert to revoke has already been revoked")
	ErrCertAlreadyExpired               = errors.New("cert to revoke has already been expired")
	ErrInvalidCertificationRevoker      = errors.New("only issuer of the cert can revoke it")
	ErrTxIsNotFromRecordOwner           = errors.New("adding record reader should be done by record owner")
	ErrAlreadyInCandidacy               = errors.New("account is already a candidate")
	ErrAlreadyVoted                     = errors.New("account has already voted for the candidate")
	ErrNotVotedYet                      = errors.New("account has not voted for anyone")
	ErrCandidateNotFound                = errors.New("candidate not found")
	ErrVotesPowerGetsMinus              = errors.New("cannot subtract a bigger value from votes power")
	ErrVotesCacheAlreadyBatching        = errors.New("votes cache is already in batch mode")
	ErrCannotConstructVotesCacheInBatch = errors.New("votes cache cannot be constructed in batch mode")
	ErrVoteDuplicate                    = errors.New("cannot vote already voted account")
	ErrDynastyExpired                   = errors.New("dynasty in the consensus state has been expired")
	ErrBlockSignatureNotExist           = errors.New("block signature does not exist in the blockheader")
	ErrTransactionSignatureNotExist     = errors.New("signature does not exist in the tx")
	ErrPayerSignatureNotExist           = errors.New("payer signature does not exist in the tx")
	ErrWrongEventTopic                  = errors.New("required event topic doesn't exist in topic list")
	ErrTransactionHashAlreadyAdded      = errors.New("transaction already added")
	ErrTypecastFailed                   = errors.New("failed to typecast")
	ErrAlreadyInVoters                  = errors.New("voter is already in voters")
	ErrNotInVoters                      = errors.New("voter is not in voters")
	ErrCannotUseZeroValue               = errors.New("value should be larger than zero")
	ErrFailedToDirectPush               = errors.New("cannot direct push to chain")
	ErrSystemError                      = errors.New("undefined system error has been occurred")
	ErrExceedBlockMaxCPUUsage           = errors.New("transaction exceeds block's max cpu usage")
	ErrExceedBlockMaxNetUsage           = errors.New("transaction exceeds block's max net usage")
	ErrCannotConvertReceipt             = errors.New("proto message cannot be converted into Receipt")
	ErrInvalidReceiptToProto            = errors.New("receipt cannot be converted into proto")
	ErrNoTransactionReceipt             = errors.New("failed to load transaction receipt")
	ErrAlreadyHaveAlias                 = errors.New("already have a alias name")
	ErrAliasAlreadyTaken                = errors.New("already occupied alias")
	ErrAliasEmptyString                 = errors.New("aliasname should not be empty string")
	ErrAliasLengthLimit                 = errors.New("aliasname should not be longer than 12 letters")
	ErrAliasInvalidChar                 = errors.New("aliasname should contain only lowercase letters and numbers")
	ErrAliasFirstLetter                 = errors.New("first letter of alias name should not be a number")
	ErrAliasNotExist                    = errors.New("doesn't have any alias")
	ErrAliasCollateralLimit             = errors.New("not enough transaction value for alias collateral")
	ErrCannotRecoverSigner              = errors.New("failed to recover payer from sign")
	ErrCannotRecoverPayer               = errors.New("failed to recover payer from payer sign")
	ErrWrongReceipt                     = errors.New("transaction receipt is wrong in block data")
	ErrInvalidCPURef                    = errors.New("invalid cpu reference")
	ErrInvalidNetRef                    = errors.New("invalid Net reference")
	ErrInvalidCPUUsage                  = errors.New("block uses too much cpu bandwidth")
	ErrInvalidNetUsage                  = errors.New("block ueses too much net bandwidth")
	ErrWrongCPUUsage                    = errors.New("block cpu usage is not matched with sum of tx cpu usage")
	ErrWrongNetUsage                    = errors.New("block net usage is not matched with sum of tx net usage")
)

// HashableBlock is an interface that can get its own or parent's hash.
type HashableBlock interface {
	Hash() []byte
	ParentHash() []byte
}

// Consensus is an interface of consensus model
type Consensus interface {
	NewConsensusState(dposRootBytes []byte, stor storage.Storage) (DposState, error)
	LoadConsensusState(dposRootBytes []byte, stor storage.Storage) (DposState, error)

	DynastySize() int
	MakeMintDynasty(ts int64, parent *Block) ([]common.Address, error)

	ForkChoice(bc *BlockChain) (newTail *Block)
	VerifyInterval(bd *BlockData, parent *Block) error
	VerifyProposer(b *Block) error
	FindLIB(bc *BlockChain) (newLIB *Block)
	FindMintProposer(ts int64, parent *Block) (common.Address, error)
}

//DposState is an interface for dpos state
type DposState interface {
	Clone() (DposState, error)
	Prepare() error
	BeginBatch() error
	Commit() error
	RollBack() error
	Reset() error
	Flush() error
	RootBytes() ([]byte, error)

	CandidateState() *trie.Batch
	DynastyState() *trie.Batch

	Candidates() ([]common.Address, error)
	Dynasty() ([]common.Address, error)
	InDynasty(addr common.Address) (bool, error)
	SetDynasty(dynasty []common.Address) error

	AddVotePowerToCandidate(candidateID []byte, amount *util.Uint128) error
	SubVotePowerToCandidate(candidateID []byte, amount *util.Uint128) error
}

// Event structure
type Event struct {
	Topic string
	Data  string
}

//SyncService interface for sync
type SyncService interface {
	ActiveDownload(targetHeight uint64) error
	IsDownloadActivated() bool
}

//TxFactory is a map for tx.TxType() to NewTxFunc
type TxFactory map[string]func(transaction *Transaction) (ExecutableTx, error)

//ExecutableTx interface for execute transaction on state
type ExecutableTx interface {
	Execute(b *Block) error
	Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error)
}

// TransactionPayload is an interface of transaction payload.
type TransactionPayload interface {
	FromBytes(b []byte) error
	ToBytes() ([]byte, error)
}
