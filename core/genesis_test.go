package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestNewGenesisBlock(t *testing.T) {
	genesisBlock, dynasties := test.NewTestGenesisBlock(t)

	assert.True(t, core.CheckGenesisBlock(genesisBlock))
	txs := genesisBlock.Transactions()
	initialMessage := "Genesis block of MediBloc"
	assert.Equalf(t, string(txs[0].Data()), initialMessage, "Initial tx payload should equal '%s'", initialMessage)

	sto := genesisBlock.Storage()

	accStateBatch, err := core.NewAccountStateBatch(genesisBlock.AccountsRoot().Bytes(), sto)
	assert.NoError(t, err)
	accState := accStateBatch.AccountState()

	addr := dynasties[0].Addr
	assert.NoError(t, err)
	acc, err := accState.GetAccount(addr.Bytes())
	assert.NoError(t, err)

	expectedBalance, _ := util.NewUint128FromString("1000000000")
	assert.Zerof(t, acc.Balance().Cmp(expectedBalance), "Balance of new account in genesis block should equal to %s", expectedBalance.String())
}
