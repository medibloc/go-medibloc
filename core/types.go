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

	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/common/trie"
)

// Transaction's string representation.
const (
	TxOperationSend                = ""
	TxOperationAddRecord           = "add_record"
	TxOperationVest                = "vest"
	TxOperationWithdrawVesting     = "withdraw_vesting"
	TxOperationAddCertification    = "add_certification"
	TxOperationRevokeCertification = "revoke_certification"
)

// Transaction payload type.
const (
	TxPayloadBinaryType = "binary"
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

// type of ReservedTask
const (
	RtWithdrawType = "withdraw"
)

// consts for reserved task-related values
const (
	RtWithdrawNum      = 3
	RtWithdrawInterval = int64(3000)
)

// Error types of core package.
var (
	ErrBalanceNotEnough                 = errors.New("balance is not enough")
	ErrBeginAgainInBatch                = errors.New("cannot begin with a batch task unfinished")
	ErrCannotCloneOnBatching            = errors.New("cannot clone on batching")
	ErrInvalidAmount                    = errors.New("invalid amount")
	ErrNotBatching                      = errors.New("not batching")
	ErrVestingNotEnough                 = errors.New("vesting is not enough")
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
	ErrInvalidSetTimestamp              = errors.New("cannot set timestamp to a sealed block")
	ErrBlockAlreadySealed               = errors.New("cannot seal an already sealed block")
	ErrNilArgument                      = errors.New("argument(s) is nil")
	ErrVoidTransaction                  = errors.New("nothing to do with transaction")
	ErrLargeTransactionNonce            = errors.New("transaction nonce is larger than expected")
	ErrSmallTransactionNonce            = errors.New("transaction nonce is smaller than expected")
	ErrMissingParentBlock               = errors.New("cannot find the block's parent block in storage")
	ErrBlockNotExist                    = errors.New("block not exist")
	ErrBlockNotSealed                   = errors.New("block should be sealed first to be signed")
	ErrInvalidBlockHeight               = errors.New("block height should be one block higher than the parent")
	ErrInvalidBlockAccountsRoot         = errors.New("invalid account state root hash")
	ErrInvalidBlockTxsRoot              = errors.New("invalid transactions state root hash")
	ErrInvalidBlockUsageRoot            = errors.New("invalid usage state root hash")
	ErrInvalidBlockRecordsRoot          = errors.New("invalid records state root hash")
	ErrInvalidBlockCandidacyRoot        = errors.New("invalid candidacy state root hash")
	ErrInvalidBlockCertificationRoot    = errors.New("invalid certification state root hash")
	ErrInvalidBlockReservationQueueHash = errors.New("invalid reservation queue hash")
	ErrInvalidBlockConsensusRoot        = errors.New("invalid block consensus root hash")
	ErrTooOldTransaction                = errors.New("transaction timestamp is too old")
	ErrInvalidTxPayload                 = errors.New("cannot unmarshal tx payload")
	ErrRecordAlreadyAdded               = errors.New("record hash already added")
	ErrRecordReaderAlreadyAdded         = errors.New("record reader hash already added")
	ErrCertReceivedAlreadyAdded         = errors.New("hash of received cert already added")
	ErrCertIssuedAlreadyAdded           = errors.New("hash of issued cert already added")
	ErrCertAlreadyRevoked               = errors.New("cert to revoke has already been revoked")
	ErrInvalidCertificationRevoker      = errors.New("only issuer of the cert can revoke it")
	ErrTxIsNotFromRecordOwner           = errors.New("adding record reader should be done by record owner")
	ErrCannotConvertResevedTask         = errors.New("proto message cannot be converted into ResevedTask")
	ErrCannotConvertResevedTasks        = errors.New("proto message cannot be converted into ResevedTasks")
	ErrInvalidReservationQueueHash      = errors.New("hash of reservation queue invalid")
	ErrReservationQueueNotBatching      = errors.New("reservation queue is not in batch mode")
	ErrReservationQueueAlreadyBatching  = errors.New("reservation queue is already in batch mode")
	ErrReservedTaskNotProcessed         = errors.New("there are reservation task(s) to be processed in the block")
	ErrInvalidReservedTaskType          = errors.New("type of reserved task is invalid")
	ErrAlreadyInCandidacy               = errors.New("account is already a candidate")
	ErrAlreadyVoted                     = errors.New("account has already voted for the candidate")
	ErrNotVotedYet                      = errors.New("account has not voted for anyone")
	ErrCandidateNotFound                = errors.New("candidate not found")
	ErrVotesPowerGetsMinus              = errors.New("cannot subtract a bigger value from votes power")
	ErrVotesCacheAlreadyBatching        = errors.New("votes cache is already in batch mode")
	ErrCannotConstructVotesCacheInBatch = errors.New("votes cache cannot be constructed in batch mode")
	ErrVoteDuplicate                    = errors.New("cannot vote already voted account")
	ErrDynastyExpired                   = errors.New("dynasty in the consensus state has been expired")
	ErrPayerSignatureNotExist           = errors.New("payer signature does not exist in the tx")
	ErrWrongEventTopic                  = errors.New("required event topic doesn't exist in topic list")
	ErrTransactionHashAlreadyAdded      = errors.New("transaction already added")
)

// HashableBlock is an interface that can get its own or parent's hash.
type HashableBlock interface {
	Hash() []byte
	ParentHash() []byte
}

// Serializable interface for serializing/deserializing
type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

// Consensus is an interface of consensus model.
type Consensus interface {
	NewConsensusState(dposRootBytes []byte, stor storage.Storage) (DposState, error)
	LoadConsensusState(dposRootBytes []byte, stor storage.Storage) (DposState, error)

	ForkChoice(bc *BlockChain) (newTail *Block)
	VerifyInterval(bd *BlockData, parent *Block) error
	VerifyProposer(bd *BlockData, parent *Block) error
	FindLIB(bc *BlockChain) (newLIB *Block)
}

type DposState interface {
	Clone() (DposState, error)
	BeginBatch() error
	Commit() error
	RollBack() error
	RootBytes() ([]byte, error)

	CandidateState() *trie.Batch
	DynastyState() *trie.Batch
}

// Event structure
type Event struct {
	Topic string
	Data  string
}

//SyncService interface for sync
type SyncService interface {
	ActiveDownload() error
}

//TxFactory is a map for tx.Type() to NewTxFunc
type TxFactory map[string]func(transaction *Transaction) (ExecutableTx, error)

//ExecutableTx interface for execute transaction on state
type ExecutableTx interface {
	Execute(b *Block) error
}
