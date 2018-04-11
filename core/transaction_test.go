package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/bytes"
	"github.com/stretchr/testify/assert"
)

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

func TestTransaction_VerifyIntegrity(t *testing.T) {
	testCount := 3
	type testTx struct {
		name    string
		tx      *core.Transaction
		privKey signature.PrivateKey
		count   int
	}

	tests := []testTx{}
	ks := keystore.NewKeyStore()

	for index := 0; index < testCount; index++ {

		from := mockAddress(t, ks)
		to := mockAddress(t, ks)

		key1, err := ks.GetKey(from)
		assert.NoError(t, err)

		tx, err := core.NewTransaction(chainID, from, to, util.Uint128Zero(), 1, core.TxPayloadBinaryType, []byte("datadata"))
		assert.NoError(t, err)

		sig, err := crypto.NewSignature(algorithm.SECP256K1)
		assert.NoError(t, err)
		sig.InitSign(key1)
		assert.NoError(t, tx.SignThis(sig))
		test := testTx{string(index), tx, key1, 1}
		tests = append(tests, test)
	}
	for _, tt := range tests {
		for index := 0; index < tt.count; index++ {
			t.Run(tt.name, func(t *testing.T) {
				signature, err := crypto.NewSignature(algorithm.SECP256K1)
				assert.NoError(t, err)
				signature.InitSign(tt.privKey)
				err = tt.tx.SignThis(signature)
				if err != nil {
					t.Errorf("Sign() error = %v", err)
					return
				}
				err = tt.tx.VerifyIntegrity(chainID)
				if err != nil {
					t.Errorf("verify failed:%s", err)
					return
				}
			})
		}
	}
}

func TestRegisterWriteKey(t *testing.T) {
	writer := "03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"
	payload := core.NewRegisterWriterPayload(common.HexToAddress(writer))
	payloadBuf, err := payload.ToBytes()
	assert.NoError(t, err)
	tx, err := core.NewTransaction(chainID,
		common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationRegisterWKey, payloadBuf)

	privKey, err := secp256k1.NewPrivateKeyFromHex("bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, tx.SignThis(sig))

	genesisState, err := genesisBlock.State().Clone()
	assert.NoError(t, err)
	genesisState.BeginBatch()
	assert.NoError(t, tx.ExecuteOnState(genesisState))
	genesisState.Commit()
	genesisState.BeginBatch()
	assert.Equal(t, tx.ExecuteOnState(genesisState), core.ErrWriterAlreadyRegistered)

	acc, err := genesisState.GetAccount(common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"))
	assert.NoError(t, err)

	assert.Equal(t, len(acc.Writers()), 1)
	assert.Equal(t, acc.Writers(), [][]byte{bytes.Hex2Bytes(writer)})

	genesisState.BeginBatch()
	assert.NoError(t, genesisState.AcceptTransaction(tx, genesisBlock.Timestamp()))
	genesisState.Commit()

	removePayload := core.NewRemoveWriterPayload(common.HexToAddress(writer))
	removePayloadBuf, err := removePayload.ToBytes()
	assert.NoError(t, err)
	txRemove, err := core.NewTransaction(chainID,
		common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
		common.Address{},
		util.Uint128Zero(), 2,
		core.TxOperationRemoveWKey, removePayloadBuf)
	assert.NoError(t, txRemove.SignThis(sig))

	genesisState.BeginBatch()
	assert.NoError(t, txRemove.ExecuteOnState(genesisState))
	genesisState.Commit()

	acc, err = genesisState.GetAccount(common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"))
	assert.NoError(t, err)

	assert.Equal(t, len(acc.Writers()), 0)
	genesisState.BeginBatch()
	assert.Equal(t, txRemove.ExecuteOnState(genesisState), core.ErrWriterNotFound)
}

func TestVerifyDelegation(t *testing.T) {
	writer := "03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"
	payload := core.NewRegisterWriterPayload(common.HexToAddress(writer))
	payloadBuf, err := payload.ToBytes()
	assert.NoError(t, err)
	owner := "03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"
	txRegister, err := core.NewTransaction(chainID,
		common.HexToAddress(owner),
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationRegisterWKey, payloadBuf)

	txDelegated, err := core.NewTransaction(chainID,
		common.HexToAddress(owner),
		common.HexToAddress("027064502450c5192fabba2ac9714606edd724a9b03cc79d26327159435eba99a7"),
		util.NewUint128FromUint(10), 2,
		core.TxPayloadBinaryType, []byte{})
	assert.NoError(t, err)

	writerPrivKey, err := secp256k1.NewPrivateKeyFromHex("b108356a113edaaf537b6cd4f506f72787d69de3c3465adc30741653949e2173")
	assert.NoError(t, err)
	sigW, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sigW.InitSign(writerPrivKey)
	assert.NoError(t, txDelegated.SignThis(sigW))

	privKey, err := secp256k1.NewPrivateKeyFromHex("bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, txRegister.SignThis(sig))

	genesisState, err := genesisBlock.State().Clone()
	assert.NoError(t, err)

	assert.Equal(t, core.ErrInvalidTxDelegation, txDelegated.VerifyDelegation(genesisState))

	genesisState.BeginBatch()
	assert.NoError(t, txRegister.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txRegister, genesisBlock.Timestamp()))
	genesisState.Commit()

	assert.NoError(t, txDelegated.VerifyDelegation(genesisState))
}
