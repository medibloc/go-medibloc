package txfactory

import (
	"github.com/medibloc/go-medibloc/core"
	corestate "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/util"
)

type Factory struct {
}

func (f *Factory) Genesis(chainID uint32, nonce uint64, msg string) *corestate.Transaction {
	template := corestate.NewTransactionTemplate(&corestate.TransactionTemplateParam{
		TxType:  transaction.TxOpGenesis,
		To:      nil,
		Value:   util.Uint128Zero(),
		Nonce:   nonce,
		ChainID: chainID,
		Payload: nil,
		From:    core.GenesisCoinbase,
		Payer:   nil,
	})

	// TODO

	return template
}
