package core

import (
	"errors"
)

const (
	TxPayloadBinaryType = "binary"
)

var (
	ErrCannotConvertTransaction  = errors.New("proto message cannot be converted into Transaction")
	ErrDuplicatedBlock           = errors.New("duplicated block")
	ErrInvalidTransactionHash    = errors.New("invalid transaction hash")
	ErrInvalidTransactionSigner  = errors.New("transaction recover public key address not equal to from")
	ErrInvalidProtoToBlock       = errors.New("protobuf message cannot be converted into Block")
	ErrInvalidProtoToBlockHeader = errors.New("protobuf message cannot be converted into BlockHeader")
	ErrInvalidChainID            = errors.New("invalid transaction chainID")
	ErrTransactionHashFailed     = errors.New("failed to hash transaction")
	ErrInvalidBlockToProto       = errors.New("block cannot be converted into proto")
	ErrInvalidSetTimestamp       = errors.New("cannot set timestamp to a sealed block")
	ErrBlockAlreadySealed        = errors.New("cannot seal an already sealed block")
	ErrNilArgument               = errors.New("argument(s) is nil")
)

// BlockPool interface
type BlockPool interface {
	FindChildren(block *Block) (childBlocks []*Block)
	FindUnlinkedAncestor(block *Block) (ancestorBlock *Block)
	FindParent(block *Block) (parentBlock *Block)
	Has(block *Block) bool
	Push(block *Block) error
	Remove(block *Block)
}
