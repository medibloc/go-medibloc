package txfactory

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/util"
)

// GenesisNonce is a nonce of genesis transaction.
const GenesisNonce = 1

// Factory is a factory for creating transactions.
type Factory struct {
	chainID uint32
}

// New creates new factory.
func New(chainID uint32) *Factory {
	return &Factory{
		chainID: chainID,
	}
}

// Genesis returns a genesis transaction.
func (f *Factory) Genesis(msg string) (*core.Transaction, error) {
	payload, err := (&transaction.DefaultPayload{
		Message: msg,
	}).ToBytes()
	if err != nil {
		return nil, err
	}

	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpGenesis,
		To:      common.Address{},
		Value:   util.Uint128Zero(),
		Nonce:   GenesisNonce,
		ChainID: f.chainID,
		Payload: payload,
	})

	if err := tx.SignGenesis(); err != nil {
		return nil, err
	}

	return tx, nil
}

// GenesisDistribution returns a genesis distribution transaction.
func (f *Factory) GenesisDistribution(to common.Address, amount *util.Uint128, nonce uint64) (*core.Transaction, error) {
	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpGenesisDistribution,
		To:      to,
		Value:   amount,
		Nonce:   nonce,
		ChainID: f.chainID,
		Payload: nil,
	})

	if err := tx.SignGenesis(); err != nil {
		return nil, err
	}
	return tx, nil
}

// Stake returns a stake transaction.
func (f *Factory) Stake(key signature.PrivateKey, amount *util.Uint128, nonce uint64) (*core.Transaction, error) {
	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpStake,
		To:      common.Address{},
		Value:   amount,
		Nonce:   nonce,
		ChainID: f.chainID,
		Payload: nil,
	})

	if err := tx.SignThis(key); err != nil {
		return nil, err
	}
	return tx, nil
}

// RegisterAlias returns a register alias transaction.
func (f *Factory) RegisterAlias(key signature.PrivateKey, collateral *util.Uint128, alias string, nonce uint64) (*core.Transaction, error) {
	payload, err := (&transaction.RegisterAliasPayload{
		AliasName: alias,
	}).ToBytes()
	if err != nil {
		return nil, err
	}
	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpRegisterAlias,
		To:      common.Address{},
		Value:   collateral,
		Nonce:   nonce,
		ChainID: f.chainID,
		Payload: payload,
	})

	if err := tx.SignThis(key); err != nil {
		return nil, err
	}
	return tx, nil
}

// BecomeCandidate returns a become candidate transaction.
func (f *Factory) BecomeCandidate(key signature.PrivateKey, collateral *util.Uint128, url string, nonce uint64) (*core.Transaction, error) {
	payload, err := (&transaction.BecomeCandidatePayload{
		URL: url,
	}).ToBytes()
	if err != nil {
		return nil, err
	}
	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpBecomeCandidate,
		To:      common.Address{},
		Value:   collateral,
		Nonce:   nonce,
		ChainID: f.chainID,
		Payload: payload,
	})

	if err := tx.SignThis(key); err != nil {
		return nil, err
	}
	return tx, nil
}

// Vote returns a vote transaction.
func (f *Factory) Vote(key signature.PrivateKey, candidateIDs [][]byte, nonce uint64) (*core.Transaction, error) {
	payload, err := (&transaction.VotePayload{
		CandidateIDs: candidateIDs,
	}).ToBytes()
	if err != nil {
		return nil, err
	}
	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpVote,
		To:      common.Address{},
		Value:   util.Uint128Zero(),
		Nonce:   nonce,
		ChainID: f.chainID,
		Payload: payload,
	})

	if err := tx.SignThis(key); err != nil {
		return nil, err
	}
	return tx, nil
}

// Transfer returns a transfer transaction.
func (f *Factory) Transfer(key signature.PrivateKey, to common.Address, amount *util.Uint128, msg string, nonce uint64) (*core.Transaction, error) {
	payload, err := (&transaction.DefaultPayload{
		Message: msg,
	}).ToBytes()
	if err != nil {
		return nil, err
	}
	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpTransfer,
		To:      to,
		Value:   amount,
		Nonce:   nonce,
		ChainID: f.chainID,
		Payload: payload,
	})

	if err := tx.SignThis(key); err != nil {
		return nil, err
	}
	return tx, nil
}
