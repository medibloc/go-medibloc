package core

import (
	"errors"
)

const (
	TxPayloadBinaryType = "binary"
)

var (
	ErrCannotConvertTransaction = errors.New("proto message cannot be converted into Transaction")
	ErrInvalidTransactionHash   = errors.New("invalid transaction hash")
	ErrInvalidTransactionSigner = errors.New("transaction recover public key address not equal to from")
	ErrInvalidChainID           = errors.New("invalid transaction chainID")
	ErrTransactionHashFailed    = errors.New("failed to hash transaction")
)
