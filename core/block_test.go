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
)

func init() {
	conf, _ := core.LoadGenesisConf(defaultGenesisConfPath)
	genesisBlock, _ = core.NewGenesisBlock(conf, blockTestDataDir)
	chainID = conf.Meta.ChainId
}

func TestNewBlock(t *testing.T) {
	assert.NotNil(t, genesisBlock)

	coinbase := common.HexToAddress("ca2d63635cae0d8e307ff9ae4af169ee103492dc")
	_, err := core.NewBlock(chainID, coinbase, genesisBlock)
	assert.NoError(t, err)
}

func TestSendExecution(t *testing.T) {
	coinbase := common.HexToAddress("ca2d63635cae0d8e307ff9ae4af169ee103492dc")
	newBlock, err := core.NewBlock(chainID, coinbase, genesisBlock)
	assert.NoError(t, err)

	privHexes := []string{
		"7f1947c1b4a6dbec0606993ef33266a17afbca07a19cc0e38b40e7dc6dede893",
		"a073b44d582ec1128f0fd2b172ff506908e43d1a036557630ac87828456020a7",
		"83835554d63e35faebb4d5d2bc7f8208ca7c6d6f4a055a62125873fb6f50726b",
	}

	privKeys := make([]signature.PrivateKey, len(privHexes))

	for i, privHex := range privHexes {
		privKeys[i], err = secp256k1.NewPrivateKeyFromHex(privHex)
		assert.NoError(t, err)
	}

	cases := []struct {
		from                 common.Address
		privKey              signature.PrivateKey
		to                   common.Address
		amount               *util.Uint128
		expectedResultAmount *util.Uint128
	}{
		{
			common.HexToAddress("0a7bad9acdfc5770a0239f8fd8939946c3b7f7f4"),
			privKeys[0],
			common.HexToAddress("fff4183e08bf9c2b38b490582fb9398cdcd875b0"),
			util.NewUint128FromUint(10),
			util.NewUint128FromUint(1000000090),
		},
		{
			common.HexToAddress("fff4183e08bf9c2b38b490582fb9398cdcd875b0"),
			privKeys[1],
			common.HexToAddress("87eb7f380a02f7a451188524f8d83b6107b4c573"),
			util.NewUint128FromUint(20),
			util.NewUint128FromUint(999999990),
		},
		{
			common.HexToAddress("87eb7f380a02f7a451188524f8d83b6107b4c573"),
			privKeys[2],
			common.HexToAddress("0a7bad9acdfc5770a0239f8fd8939946c3b7f7f4"),
			util.NewUint128FromUint(100),
			util.NewUint128FromUint(999999920),
		},
	}

	txs := make(core.Transactions, len(cases))
	signers := make([]signature.Signature, len(cases))

	for i, c := range cases {
		txs[i], err = core.NewTransaction(chainID, c.from, c.to, c.amount, 1, core.TxPayloadBinaryType, []byte{})
		assert.NoError(t, err)

		signers[i], err = crypto.NewSignature(algorithm.SECP256K1)
		assert.NoError(t, err)
		signers[i].InitSign(c.privKey)
		assert.NoError(t, txs[i].SignThis(signers[i]))
	}

	newBlock.SetTransactions(txs)

	newBlock.BeginBatch()
	assert.NoError(t, newBlock.ExecuteAll())
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

	for _, c := range cases {
		acc, err := accState.GetAccount(c.from.Bytes())
		assert.NoError(t, err)

		assert.Zero(t, acc.Balance().Cmp(c.expectedResultAmount))
	}

	assert.NoError(t, newBlock.VerifyIntegrity())
}

func TestSendMoreThanBalance(t *testing.T) {
	coinbase := common.HexToAddress("ca2d63635cae0d8e307ff9ae4af169ee103492dc")
	newBlock, err := core.NewBlock(chainID, coinbase, genesisBlock)
	assert.NoError(t, err)

	privHex := "7f1947c1b4a6dbec0606993ef33266a17afbca07a19cc0e38b40e7dc6dede893"

	privKey, err := secp256k1.NewPrivateKeyFromHex(privHex)
	assert.NoError(t, err)

	from := common.HexToAddress("87eb7f380a02f7a451188524f8d83b6107b4c573")
	to := common.HexToAddress("0a7bad9acdfc5770a0239f8fd8939946c3b7f7f4")

	balance := util.NewUint128FromUint(1000000090)
	sendingAmount, err := balance.Add(util.NewUint128FromUint(1))

	tx, err := core.NewTransaction(chainID, from, to, sendingAmount, 1, core.TxPayloadBinaryType, []byte{})
	assert.NoError(t, err)

	signer, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	signer.InitSign(privKey)
	assert.NoError(t, tx.SignThis(signer))

	blockState := newBlock.State()

	newBlock.BeginBatch()
	assert.Equal(t, blockState.ExecuteTx(tx), core.ErrBalanceNotEnough)
	newBlock.RollBack()
}
