package core_test

import (
  "encoding/hex"
  "testing"

  "github.com/medibloc/go-medibloc/core"
  "github.com/medibloc/go-medibloc/util"
  "github.com/stretchr/testify/assert"
)

var (
  defaultGenesisConfPath = "../conf/default/genesis.conf"
  genesisTestDataDir     = "./testdata/genesis"

  initialDist1 = "ff7b1d22d234bde673bfa783d6c0c6b835aab407"
)

func TestNewGenesisBlock(t *testing.T) {
  conf, err := core.LoadGenesisConf(defaultGenesisConfPath)
  assert.NoError(t, err)
  genesisBlock, err := core.NewGenesisBlock(conf, genesisTestDataDir)
  assert.NoError(t, err)
  assert.True(t, core.CheckGenesisBlock(genesisBlock))
  txs := genesisBlock.Transactions()
  initialMessage := "Genesis block of MediBloc"
  assert.Equalf(t, string(txs[0].Data()), initialMessage, "Initial tx payload should equal '%s'", initialMessage)

  sto := genesisBlock.Storage()

  accStateBatch, err := core.NewAccountStateBatch(genesisBlock.AccountsRoot().Bytes(), sto)
  assert.NoError(t, err)
  accState := accStateBatch.AccountState()

  addr, err := hex.DecodeString("ff7b1d22d234bde673bfa783d6c0c6b835aab407")
  assert.NoError(t, err)
  acc, err := accState.GetAccount(addr)
  assert.NoError(t, err)

  expectedBalance, _ := util.NewUint128FromString("1000000000")
  assert.Zerof(t, acc.Balance().Cmp(expectedBalance), "Balance of new account in genesis block should equal to %s", expectedBalance.String())
}
