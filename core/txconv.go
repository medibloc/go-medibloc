package core

var txFactory TxFactory

// Inject injects TxFactory dependencies.
func InjectTxFactory(txFactoryj TxFactory) {
	txFactory = txFactory
}

// TxConv returns executable tx.
func TxConv(tx *Transaction) (ExecutableTx, error) {
	return txFactory.Executable(tx)
}

// TxFactory is a map for Transaction to ExecutableTx
type TxFactory interface {
	Executable(tx *Transaction) (ExecutableTx, error)
}

type MapTxFactory map[string]func(tx *Transaction) (ExecutableTx, error)

func (m MapTxFactory) Executable(tx *Transaction) (ExecutableTx, error) {
	constructor, ok := m[tx.TxType()]
	if !ok {
		return nil, ErrInvalidTransactionType
	}
	return constructor(tx)
}
