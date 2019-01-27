package txfactory

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/util"
)

const GenesisNonce = 1

type Factory struct {
	chainID uint32
}

func NewFactory(chainID uint32) *Factory {
	return &Factory{
		chainID: chainID,
	}
}

func (f *Factory) Genesis(msg string) (*core.Transaction, error) {
	payload := transaction.DefaultPayload{
		Message: msg,
	}
	payloadBytes, err := payload.ToBytes()
	if err != nil {
		return nil, err
	}

	tx := core.NewTransactionTemplate(&core.TransactionTemplateParam{
		TxType:  transaction.TxOpGenesis,
		To:      nil,
		Value:   util.Uint128Zero(),
		Nonce:   GenesisNonce,
		ChainID: f.chainID,
		Payload: payloadBytes,
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
