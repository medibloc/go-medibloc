package core

import (
	"errors"

	"github.com/medibloc/go-medibloc/common"
)

const (
	TxOperationSend         = ""
	TxOperationAddRecord    = "add_record"
	TxOperationDeposit      = "deposit"
	TxOperationWidthdraw    = "widthdraw"
	TxOperationRegisterRKey = "register_rkey"
	TxOperationRegisterWKey = "register_wkey"
	TxOperationRemoveWKey   = "remove_wkey"
)

const (
	TxPayloadBinaryType = "binary"
)

// MessageType from network.
const (
	MessageTypeNewTx = "newtx"
)

// Error types of core package.
var (
	ErrCannotConvertTransaction  = errors.New("proto message cannot be converted into Transaction")
	ErrDuplicatedBlock           = errors.New("duplicated block")
	ErrDuplicatedTransaction     = errors.New("duplicated transaction")
	ErrInvalidTransactionHash    = errors.New("invalid transaction hash")
	ErrInvalidTransactionSigner  = errors.New("transaction recover public key address not equal to from")
	ErrInvalidProtoToBlock       = errors.New("protobuf message cannot be converted into Block")
	ErrInvalidProtoToBlockHeader = errors.New("protobuf message cannot be converted into BlockHeader")
	ErrInvalidChainID            = errors.New("invalid transaction chainID")
	ErrTransactionHashFailed     = errors.New("failed to hash transaction")
	ErrInvalidBlockToProto       = errors.New("block cannot be converted into proto")
	ErrInvalidBlockHash          = errors.New("invalid block hash")
	ErrInvalidSetTimestamp       = errors.New("cannot set timestamp to a sealed block")
	ErrBlockAlreadySealed        = errors.New("cannot seal an already sealed block")
	ErrNilArgument               = errors.New("argument(s) is nil")
	ErrVoidTransaction           = errors.New("nothing to do with transaction")
	ErrLargeTransactionNonce     = errors.New("transaction nonce is larger than expected")
	ErrSmallTransactionNonce     = errors.New("transaction nonce is smaller than expected")
	ErrBlockNotSealed            = errors.New("block should be sealed first to be signed")
	ErrInvalidBlockAccountsRoot  = errors.New("invalid account state root hash")
	ErrInvalidBlockTxsRoot       = errors.New("invalid transactions state root hash")
	ErrTooOldTransaction         = errors.New("transaction timestamp is too old")
	ErrWriterAlreadyRegistered   = errors.New("writer address already registered")
	ErrWriterNotFound            = errors.New("writer to remove not found")
	ErrInvalidTxPayload          = errors.New("cannot unmarshal tx payload")
	ErrInvalidTxDelegation       = errors.New("tx signer is not owner or one of writers")
)

// HashableBlock is an interface that can get its own or parent's hash.
type HashableBlock interface {
	Hash() common.Hash
	ParentHash() common.Hash
}
