package core

import (
	"errors"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
)

// Transaction's string representation.
const (
	TxOperationSend             = ""
	TxOperationAddRecord        = "add_record"
	TxOperationAddRecordReader  = "add_record_reader"
	TxOperationVest             = "vest"
	TxOperationWidthdrawVesting = "widthdraw_vesting"
	TxOperationRegisterWKey     = "register_wkey"
	TxOperationRemoveWKey       = "remove_wkey"
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

// Error types of core package.
var (
	ErrCannotConvertTransaction        = errors.New("proto message cannot be converted into Transaction")
	ErrDuplicatedBlock                 = errors.New("duplicated block")
	ErrDuplicatedTransaction           = errors.New("duplicated transaction")
	ErrInvalidTransactionHash          = errors.New("invalid transaction hash")
	ErrInvalidTransactionSigner        = errors.New("transaction recover public key address not equal to from")
	ErrInvalidProtoToBlock             = errors.New("protobuf message cannot be converted into Block")
	ErrInvalidProtoToBlockHeader       = errors.New("protobuf message cannot be converted into BlockHeader")
	ErrInvalidChainID                  = errors.New("invalid transaction chainID")
	ErrTransactionHashFailed           = errors.New("failed to hash transaction")
	ErrInvalidBlockToProto             = errors.New("block cannot be converted into proto")
	ErrInvalidBlockHash                = errors.New("invalid block hash")
	ErrInvalidSetTimestamp             = errors.New("cannot set timestamp to a sealed block")
	ErrBlockAlreadySealed              = errors.New("cannot seal an already sealed block")
	ErrNilArgument                     = errors.New("argument(s) is nil")
	ErrVoidTransaction                 = errors.New("nothing to do with transaction")
	ErrLargeTransactionNonce           = errors.New("transaction nonce is larger than expected")
	ErrSmallTransactionNonce           = errors.New("transaction nonce is smaller than expected")
	ErrMissingParentBlock              = errors.New("cannot find the block's parent block in storage")
	ErrBlockNotExist                   = errors.New("block not exist")
	ErrBlockNotSealed                  = errors.New("block should be sealed first to be signed")
	ErrInvalidBlockAccountsRoot        = errors.New("invalid account state root hash")
	ErrInvalidBlockTxsRoot             = errors.New("invalid transactions state root hash")
	ErrInvalidBlockConsensusRoot       = errors.New("invalid block consensus root hash")
	ErrTooOldTransaction               = errors.New("transaction timestamp is too old")
	ErrWriterAlreadyRegistered         = errors.New("writer address already registered")
	ErrWriterNotFound                  = errors.New("writer to remove not found")
	ErrInvalidTxPayload                = errors.New("cannot unmarshal tx payload")
	ErrInvalidTxDelegation             = errors.New("tx signer is not owner or one of writers")
	ErrRecordAlreadyAdded              = errors.New("record hash already added")
	ErrRecordReaderAlreadyAdded        = errors.New("record reader hash already added")
	ErrTxIsNotFromRecordOwner          = errors.New("adding record reader should be done by record owner")
	ErrCannotConvertResevedTask        = errors.New("proto message cannot be converted into ResevedTask")
	ErrCannotConvertResevedTasks       = errors.New("proto message cannot be converted into ResevedTasks")
	ErrInvalidReservationQueueHash     = errors.New("hash of reservation queue invalid")
	ErrReservationQueueNotBatching     = errors.New("reservation queue is not in batch mode")
	ErrReservationQueueAlreadyBatching = errors.New("reservation queue is already in batch mode")
)

// HashableBlock is an interface that can get its own or parent's hash.
type HashableBlock interface {
	Hash() common.Hash
	ParentHash() common.Hash
}

// Medlet interface for component discovery.
type Medlet interface {
	Config() *medletpb.Config
	Storage() storage.Storage
	Genesis() *corepb.Genesis
	NetService() net.Service
}
