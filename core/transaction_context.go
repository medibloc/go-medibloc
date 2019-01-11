package core

import (
	"time"

	corestate "github.com/medibloc/go-medibloc/core/state"
)

// TxContext struct represents a transaction with it's context.
type TxContext struct {
	*corestate.Transaction
	exec       ExecutableTx
	incomeTime time.Time
	broadcast  bool
}

// NewTxContext returns TxContext.
func NewTxContext(tx *corestate.Transaction) (*TxContext, error) {
	return newTxContext(tx, false)
}

// NewTxContextWithBroadcast set broadcast flag and returns TxContext.
func NewTxContextWithBroadcast(tx *corestate.Transaction) (*TxContext, error) {
	return newTxContext(tx, true)
}

func newTxContext(tx *corestate.Transaction, broadcast bool) (*TxContext, error) {
	exec, err := TxConv(tx)
	if err != nil {
		return nil, err
	}
	return &TxContext{
		Transaction: tx,
		exec:        exec,
		incomeTime:  time.Now(),
		broadcast:   broadcast,
	}, nil
}
