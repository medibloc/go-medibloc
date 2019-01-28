package txfactory

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/util"
)

const GenesisNonce = 1

type Factory struct {
	chainID uint32
}

func New(chainID uint32) *Factory {
	return &Factory{
		chainID: chainID,
	}
}

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
