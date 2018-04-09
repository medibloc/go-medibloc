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

func TestNewBlock(t *testing.T) {
	assert.NotNil(t, genesisBlock)

	coinbase := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	_, err := core.NewBlock(chainID, coinbase, genesisBlock)
	assert.NoError(t, err)
}

func TestSendExecution(t *testing.T) {
	coinbase := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	newBlock, err := core.NewBlock(chainID, coinbase, genesisBlock)
	assert.NoError(t, err)

	privHexes := []string{
		"bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0",
		"b108356a113edaaf537b6cd4f506f72787d69de3c3465adc30741653949e2173",
		"a396832a6aac41cb5844ecf86b2ec0406daa5aad4a6612964aa5e8c65abdf451",
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
			common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
			privKeys[0],
			common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"),
			util.NewUint128FromUint(10),
			util.NewUint128FromUint(1000000090),
		},
		{
			common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"),
			privKeys[1],
			common.HexToAddress("027064502450c5192fabba2ac9714606edd724a9b03cc79d26327159435eba99a7"),
			util.NewUint128FromUint(20),
			util.NewUint128FromUint(999999990),
		},
		{
			common.HexToAddress("027064502450c5192fabba2ac9714606edd724a9b03cc79d26327159435eba99a7"),
			privKeys[2],
			common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
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

	coinbaseKey, err := secp256k1.HexToECDSA("ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea")

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
	coinbase := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	newBlock, err := core.NewBlock(chainID, coinbase, genesisBlock)
	assert.NoError(t, err)

	privHex := "ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea"

	privKey, err := secp256k1.NewPrivateKeyFromHex(privHex)
	assert.NoError(t, err)

	from := common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939")
	to := common.HexToAddress("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21")

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
