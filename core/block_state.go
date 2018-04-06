package core

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
)

type states struct {
	accState *AccountStateBatch
	txsState *TrieBatch

	storage storage.Storage
}

func newStates(stor storage.Storage) (*states, error) {
	accState, err := NewAccountStateBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	txsState, err := NewTrieBatch(nil, stor)
	if err != nil {
		return nil, err
	}

	return &states{
		accState: accState,
		txsState: txsState,
		storage:  stor,
	}, nil
}

func (st *states) Clone() (*states, error) {
	accState, err := NewAccountStateBatch(st.accState.RootHash(), st.storage)
	if err != nil {
		return nil, err
	}

	txsState, err := NewTrieBatch(st.txsState.RootHash(), st.storage)
	if err != nil {
		return nil, err
	}

	return &states{
		accState: accState,
		txsState: txsState,
		storage:  st.storage,
	}, nil
}

func (st *states) BeginBatch() error {
	if err := st.accState.BeginBatch(); err != nil {
		return err
	}
	if err := st.txsState.BeginBatch(); err != nil {
		return err
	}
	return nil
}

func (st *states) RollBack() error {
	if err := st.accState.RollBack(); err != nil {
		return err
	}
	if err := st.txsState.RollBack(); err != nil {
		return err
	}
	return nil
}

func (st *states) Commit() error {
	if err := st.accState.Commit(); err != nil {
		return err
	}
	if err := st.txsState.Commit(); err != nil {
		return err
	}
	return nil
}

func (st *states) AccountsRoot() common.Hash {
	return common.BytesToHash(st.accState.RootHash())
}

func (st *states) TransactionsRoot() common.Hash {
	return common.BytesToHash(st.txsState.RootHash())
}

func (st *states) LoadAccountsRoot(rootHash common.Hash) error {
	accState, err := NewAccountStateBatch(rootHash.Bytes(), st.storage)
	if err != nil {
		return err
	}
	st.accState = accState
	return nil
}

func (st *states) LoadTxssRoot(rootHash common.Hash) error {
	txsState, err := NewTrieBatch(rootHash.Bytes(), st.storage)
	if err != nil {
		return err
	}
	st.txsState = txsState
	return nil
}

func (st *states) GetAccount(address common.Address) (Account, error) {
	return st.accState.GetAccount(address.Bytes())
}

func (st *states) AddBalance(address common.Address, amount *util.Uint128) error {
	return st.accState.AddBalance(address.Bytes(), amount)
}

func (st *states) SubBalance(address common.Address, amount *util.Uint128) error {
	return st.accState.SubBalance(address.Bytes(), amount)
}

func (st *states) IncrementNonce(address common.Address) error {
	return st.accState.IncrementNonce(address.Bytes())
}

func (st *states) GetTx(txHash common.Hash) ([]byte, error) {
	return st.txsState.Get(txHash.Bytes())
}

func (st *states) PutTx(txHash common.Hash, txBytes []byte) error {
	return st.txsState.Put(txHash.Bytes(), txBytes)
}

// BlockState possesses every states a block should have
type BlockState struct {
	*states
	snapshot *states
}

// NewBlockState creates a new block state
func NewBlockState(stor storage.Storage) (*BlockState, error) {
	states, err := newStates(stor)
	if err != nil {
		return nil, err
	}
	return &BlockState{
		states:   states,
		snapshot: nil,
	}, nil
}

// Clone clones block state
func (bs *BlockState) Clone() (*BlockState, error) {
	states, err := bs.states.Clone()
	if err != nil {
		return nil, err
	}
	return &BlockState{
		states:   states,
		snapshot: nil,
	}, nil
}

// BeginBatch begins batch
func (bs *BlockState) BeginBatch() error {
	snapshot, err := bs.states.Clone()
	if err != nil {
		return err
	}
	if err := bs.states.BeginBatch(); err != nil {
		return err
	}
	bs.snapshot = snapshot
	return nil
}

// RollBack rolls back batch
func (bs *BlockState) RollBack() error {
	bs.states = bs.snapshot
	bs.snapshot = nil
	return nil
}

// Commit saves batch updates
func (bs *BlockState) Commit() error {
	if err := bs.states.Commit(); err != nil {
		return err
	}
	bs.snapshot = nil
	return nil
}
