package transaction

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
)

type GenesisTx struct {
	*core.Transaction
	hash []byte
}

var _ core.ExecutableTx = &GenesisTx{}

func NewGenesisTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(DefaultPayload)
	if err := BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	return &GenesisTx{
		Transaction: tx,
		hash:        tx.Hash(),
	}, nil
}

func (tx *GenesisTx) Execute(b *core.Block) error {
	if b.Height() != core.GenesisHeight {
		return core.ErrNotGenesisBlock
	}
	return nil
}

func (tx *GenesisTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(0, 0)
}

func (tx *GenesisTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, util.Uint128Zero()
}

func (tx *GenesisTx) RecoverFrom() (common.Address, error) {
	return recoverGenesis(tx.Transaction)
}

type GenesisDistributionTx struct {
	*core.Transaction
	target  common.Address
	balance *util.Uint128
}

var _ core.ExecutableTx = &GenesisDistributionTx{}

func NewGenesisDistributionTx(tx *core.Transaction) (core.ExecutableTx, error) {
	return &GenesisDistributionTx{
		Transaction: tx,
		target:      tx.To(),
		balance:     tx.Value(),
	}, nil
}

func (tx *GenesisDistributionTx) Execute(b *core.Block) error {
	if b.Height() != core.GenesisHeight {
		return core.ErrNotGenesisBlock
	}
	acc, err := b.State().GetAccount(tx.target)
	if err != nil {
		return err
	}
	if acc.Balance.Cmp(util.Uint128Zero()) != 0 {
		return core.ErrGenesisDistributionAllowedOnce
	}

	acc.Balance, err = acc.Balance.Add(tx.balance)
	if err != nil {
		return err
	}
	err = b.State().PutAccount(acc)
	if err != nil {
		return err
	}
	supply := b.State().Supply()
	supply, err = supply.Add(tx.balance)
	if err != nil {
		return err
	}
	b.State().SetSupply(supply)

	return nil
}

func (tx *GenesisDistributionTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(0, 0)
}

func (tx *GenesisDistributionTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, util.Uint128Zero()
}

func (tx *GenesisDistributionTx) RecoverFrom() (common.Address, error) {
	return recoverGenesis(tx.Transaction)
}
