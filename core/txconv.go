package core

var txMapper TxMapper

// TxMapper is a map for transaction to executable tx.
type TxMapper map[string]func(tx *Transaction) (ExecutableTx, error)

// Executable converts transaction to executable tx.
func (m TxMapper) Executable(tx *Transaction) (ExecutableTx, error) {
	constructor, ok := m[tx.TxType()]
	if !ok {
		return nil, ErrInvalidTransactionType
	}
	return constructor(tx)
}

// InjectTxMapper injects TxMapper dependencies.
func InjectTxMapper(mapper TxMapper) {
	txMapper = mapper
}

// TxConv returns executable tx.
func TxConv(tx *Transaction) (ExecutableTx, error) {
	return txMapper.Executable(tx)
}
