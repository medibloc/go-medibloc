package core_test

import (
  "testing"

  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/core"
  "github.com/medibloc/go-medibloc/crypto"
  "github.com/medibloc/go-medibloc/crypto/signature"
  "github.com/medibloc/go-medibloc/crypto/signature/algorithm"
  "github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
  "github.com/medibloc/go-medibloc/util"
  "github.com/stretchr/testify/assert"
)

var (
  blockTestDataDir = "./testdata/block"
  chainId          uint32
)

func init() {
  conf, _ := core.LoadGenesisConf(defaultGenesisConfPath)
  genesisBlock, _ = core.NewGenesisBlock(conf, blockTestDataDir)
  chainId = conf.Meta.ChainId
}

func TestNewBlock(t *testing.T) {
  assert.NotNil(t, genesisBlock)

  coinbase := common.HexToAddress("ca2d63635cae0d8e307ff9ae4af169ee103492dc")
  _, err := core.NewBlock(chainId, coinbase, genesisBlock)
  assert.NoError(t, err)
}

func TestSendExecution(t *testing.T) {
  coinbase := common.HexToAddress("ca2d63635cae0d8e307ff9ae4af169ee103492dc")
  newBlock, err := core.NewBlock(chainId, coinbase, genesisBlock)
  assert.NoError(t, err)

  users := []common.Address{
    common.HexToAddress("0a7bad9acdfc5770a0239f8fd8939946c3b7f7f4"),
    common.HexToAddress("fff4183e08bf9c2b38b490582fb9398cdcd875b0"),
    common.HexToAddress("87eb7f380a02f7a451188524f8d83b6107b4c573"),
  }

  privHexes := []string{
    "7f1947c1b4a6dbec0606993ef33266a17afbca07a19cc0e38b40e7dc6dede893",
    "a073b44d582ec1128f0fd2b172ff506908e43d1a036557630ac87828456020a7",
    "83835554d63e35faebb4d5d2bc7f8208ca7c6d6f4a055a62125873fb6f50726b",
  }

  privKeys := make([]signature.PrivateKey, len(privHexes))

  for i, privHex := range privHexes {
    ecdsaKey, err := secp256k1.HexToECDSA(privHex)
    assert.NoError(t, err)
    privKeys[i] = secp256k1.NewPrivateKey(ecdsaKey)
  }

  txs := make(core.Transactions, len(users))

  txs[0], err = core.NewTransaction(chainId, users[0], users[1], util.NewUint128FromUint(10), 1, core.TxPayloadBinaryType, []byte{})
  assert.NoError(t, err)
  txs[1], err = core.NewTransaction(chainId, users[1], users[2], util.NewUint128FromUint(20), 1, core.TxPayloadBinaryType, []byte{})
  assert.NoError(t, err)
  txs[2], err = core.NewTransaction(chainId, users[2], users[0], util.NewUint128FromUint(100), 1, core.TxPayloadBinaryType, []byte{})
  assert.NoError(t, err)

  signers := make([]signature.Signature, len(privKeys))

  for i, privKey := range privKeys {
    signers[i], err = crypto.NewSignature(algorithm.SECP256K1)
    assert.NoError(t, err)
    signers[i].InitSign(privKey)
  }

  assert.NoError(t, txs[0].SignThis(signers[0]))
  assert.NoError(t, txs[1].SignThis(signers[1]))
  assert.NoError(t, txs[2].SignThis(signers[2]))

  newBlock.SetTransactions(txs)

  newBlock.Begin()
  assert.NoError(t, newBlock.Execute())
  newBlock.Commit()
  assert.NoError(t, newBlock.Seal())

  coinbaseKey, err := secp256k1.HexToECDSA("4bad60a1096693414a1e7251784b5c456ab1861d3d80f02dfb7e4b86968999e2")

  blockSigner, err := crypto.NewSignature(algorithm.SECP256K1)
  assert.NoError(t, err)
  blockSigner.InitSign(secp256k1.NewPrivateKey(coinbaseKey))

  assert.NoError(t, newBlock.SignThis(blockSigner))

  assert.NoError(t, newBlock.VerifyState())

  accStateBatch, err := core.NewAccountStateBatch(newBlock.AccountsRoot().Bytes(), newBlock.Storage())
  assert.NoError(t, err)
  accState := accStateBatch.AccountState()

  acc0, err := accState.GetAccount(users[0].Bytes())
  assert.NoError(t, err)

  assert.Zero(t, acc0.Balance().Cmp(util.NewUint128FromUint(1000000090)))

  acc1, err := accState.GetAccount(users[1].Bytes())
  assert.NoError(t, err)

  assert.Zero(t, acc1.Balance().Cmp(util.NewUint128FromUint(999999990)))

  acc2, err := accState.GetAccount(users[2].Bytes())
  assert.NoError(t, err)

  assert.Zero(t, acc2.Balance().Cmp(util.NewUint128FromUint(999999920)))

  assert.NoError(t, newBlock.VerifyIntegrity())
}
