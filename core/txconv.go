package core

import (
	corestate "github.com/medibloc/go-medibloc/core/state"
)

var txFactory TxFactory

// InjectTxFactory injects TxFactory dependencies.
func InjectTxFactory(factory TxFactory) {
	txFactory = factory
}

// TxConv returns executable tx.
func TxConv(tx *corestate.Transaction) (ExecutableTx, error) {
	return txFactory.Executable(tx)
}

// TxFactory is a map for Transaction to ExecutableTx
type TxFactory interface {
	Executable(tx *corestate.Transaction) (ExecutableTx, error)
}

// MapTxFactory is a map for transaction to executable tx.
type MapTxFactory map[string]func(tx *corestate.Transaction) (ExecutableTx, error)

// Executable converts transaction to executable tx.
func (m MapTxFactory) Executable(tx *corestate.Transaction) (ExecutableTx, error) {
	constructor, ok := m[tx.TxType()]
	if !ok {
		return nil, ErrTxTypeInvalid
	}
	return constructor(tx)
}
