package core

import (
	"encoding/hex"

	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// AccountStateBatch batch for AccountState
type AccountStateBatch struct {
	as            *accountState
	batching      bool
	stageAccounts map[string]*account
	storage       storage.Storage
}

// NewAccountStateBatch create and return new AccountStateBatch instance
func NewAccountStateBatch(accountsRootHash []byte, s storage.Storage) (*AccountStateBatch, error) {
	// TODO Get initial accounts from config or another place
	accTrie, err := trie.NewTrie(accountsRootHash, s)
	if err != nil {
		return nil, err
	}
	return &AccountStateBatch{
		as:            &accountState{accounts: accTrie, storage: s},
		batching:      false,
		stageAccounts: make(map[string]*account),
		storage:       s,
	}, nil
}

// BeginBatch begin batch
func (as *AccountStateBatch) BeginBatch() error {
	if as.batching {
		return ErrBeginAgainInBatch
	}
	as.batching = true
	return nil
}

// Clone clone AccountStateBatch enable only when not batching
func (as *AccountStateBatch) Clone() (*AccountStateBatch, error) {
	if as.batching {
		return nil, ErrCannotCloneOnBatching
	}
	return NewAccountStateBatch(as.as.accounts.RootHash(), as.storage)
}

// Commit commit batch WARNING: not thread-safe
func (as *AccountStateBatch) Commit() error {
	if !as.batching {
		return ErrNotBatching
	}

	for k, acc := range as.stageAccounts {
		bytes, err := hex.DecodeString(k)
		if err != nil {
			return err
		}
		bytes, err = acc.toBytes()
		if err != nil {
			return err
		}
		as.as.accounts.Put(acc.address, bytes)
	}

	as.resetBatch()
	return nil
}

// RollBack rollback batch WARNING: not thread-safe
func (as *AccountStateBatch) RollBack() error {
	if !as.batching {
		return ErrNotBatching
	}
	as.resetBatch()
	return nil
}

func (as *AccountStateBatch) getAccount(address []byte) (*account, error) {
	s := hex.EncodeToString(address)
	if acc, ok := as.stageAccounts[s]; ok {
		return acc, nil
	}
	stageAndReturn := func(acc2 *account) *account {
		as.stageAccounts[s] = acc2
		return acc2
	}
	accBytes, err := as.as.accounts.Get(address)
	if err != nil {
		// create account not exist in
		return stageAndReturn(&account{
			address: address,
			balance: util.NewUint128(),
			vesting: util.NewUint128(),
			nonce:   0,
		}), nil
	}
	acc, err := loadAccount(accBytes, as.storage)
	if err != nil {
		return nil, err
	}
	return stageAndReturn(acc), nil
}

func (as *AccountStateBatch) resetBatch() {
	as.batching = false
	as.stageAccounts = make(map[string]*account)
}

// AddBalance add balance
func (as *AccountStateBatch) AddBalance(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	balance, err := acc.balance.Add(amount)
	if err != nil {
		return err
	}
	acc.balance = balance
	return nil
}

// AddWriter adds writer address of this account
// writers can use the account's bandwidth and add record of her
func (as *AccountStateBatch) AddWriter(address []byte, newWriter []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	for _, w := range acc.writers {
		if byteutils.Equal(newWriter, w) {
			return ErrWriterAlreadyRegistered
		}
	}
	acc.writers = append(acc.writers, newWriter)
	return nil
}

// RemoveWriter removes a writer from account's writers list
func (as *AccountStateBatch) RemoveWriter(address []byte, writer []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	writers := [][]byte{}
	for _, w := range acc.writers {
		if byteutils.Equal(writer, w) {
			continue
		}
		writers = append(writers, w)
	}
	if len(acc.writers) == len(writers) {
		return ErrWriterNotFound
	}
	acc.writers = writers
	return nil
}

// AddRecord adds a record hash in account's records list
func (as *AccountStateBatch) AddRecord(address []byte, hash []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	for _, r := range acc.records {
		if byteutils.Equal(hash, r) {
			return ErrRecordAlreadyAdded
		}
	}
	acc.records = append(acc.records, hash)
	return nil
}

// GetAccount get account in stage(batching) or in original accountState
func (as *AccountStateBatch) GetAccount(address []byte) (Account, error) {
	s := hex.EncodeToString(address)
	if acc, ok := as.stageAccounts[s]; ok {
		return acc, nil
	}
	accBytes, err := as.as.accounts.Get(address)
	if err == nil {
		acc, err := loadAccount(accBytes, as.storage)
		if err != nil {
			return nil, err
		}
		return acc, nil
	}
	return nil, ErrNotFound
}

// AccountState getter for accountState
func (as *AccountStateBatch) AccountState() AccountState {
	return as.as
}

// SubBalance subtract balance
func (as *AccountStateBatch) SubBalance(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	if amount.Cmp(acc.balance) > 0 {
		return ErrBalanceNotEnough
	}
	balance, err := acc.balance.Sub(amount)
	if err != nil {
		return err
	}
	acc.balance = balance
	return nil
}

// IncrementNonce increment account's nonce
func (as *AccountStateBatch) IncrementNonce(address []byte) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	acc.nonce++
	return nil
}

// AddVesting increases vesting
func (as *AccountStateBatch) AddVesting(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	vesting, err := acc.vesting.Add(amount)
	if err != nil {
		return err
	}
	acc.vesting = vesting
	return nil
}

// SubVesting decreases vesting
func (as *AccountStateBatch) SubVesting(address []byte, amount *util.Uint128) error {
	if !as.batching {
		return ErrNotBatching
	}
	acc, err := as.getAccount(address)
	if err != nil {
		return err
	}
	if amount.Cmp(acc.vesting) > 0 {
		return ErrVestingNotEnough
	}
	vesting, err := acc.vesting.Sub(amount)
	if err != nil {
		return err
	}
	acc.vesting = vesting
	return nil
}

// RootHash returns root hash of accounts trie
func (as *AccountStateBatch) RootHash() []byte {
	return as.as.accounts.RootHash()
}
