package core

import (
	corestate "github.com/medibloc/go-medibloc/core/state"
)

var txMapper TxMapper

// TxMap is a map for transaction to executable tx.
type TxMapper map[string]func(tx *corestate.Transaction) (ExecutableTx, error)

// Executable converts transaction to executable tx.
func (m TxMapper) Executable(tx *corestate.Transaction) (ExecutableTx, error) {
	constructor, ok := m[tx.TxType()]
	if !ok {
		return nil, ErrTxTypeInvalid
	}
	return constructor(tx)
}

// InjectTxMapper injects TxMapper dependencies.
func InjectTxMapper(mapper TxMapper) {
	txMapper = mapper
}

// TxConv returns executable tx.
func TxConv(tx *corestate.Transaction) (ExecutableTx, error) {
	return txMapper.Executable(tx)
}
