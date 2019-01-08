package core

import "time"

// TxContext struct represents a transaction with it's context.
type TxContext struct {
	*Transaction
	exec       ExecutableTx
	incomeTime time.Time
	broadcast  bool
}

// NewTxContext returns TxContext.
func NewTxContext(tx *Transaction) (*TxContext, error) {
	return newTxContext(tx, false)
}

// NewTxContextWithBroadcast set broadcast flag and returns TxContext.
func NewTxContextWithBroadcast(tx *Transaction) (*TxContext, error) {
	return newTxContext(tx, true)
}

func newTxContext(tx *Transaction, broadcast bool) (*TxContext, error) {
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
