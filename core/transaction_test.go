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
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
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

		from := test.MockAddress(t, ks)
		to := test.MockAddress(t, ks)

		key1, err := ks.GetKey(from)
		assert.NoError(t, err)

		tx, err := core.NewTransaction(test.ChainID, from, to, util.Uint128Zero(), 1, core.TxPayloadBinaryType, []byte("datadata"))
		assert.NoError(t, err)

		sig, err := crypto.NewSignature(algorithm.SECP256K1)
		assert.NoError(t, err)
		sig.InitSign(key1)
		assert.NoError(t, tx.SignThis(sig))
		tests = append(tests, testTx{string(index), tx, key1, 1})
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
				err = tt.tx.VerifyIntegrity(test.ChainID)
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
	tx, err := core.NewTransaction(test.ChainID,
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

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)
	genesisState.BeginBatch()
	assert.NoError(t, tx.ExecuteOnState(genesisState))
	genesisState.Commit()
	genesisState.BeginBatch()
	assert.Equal(t, tx.ExecuteOnState(genesisState), core.ErrWriterAlreadyRegistered)

	acc, err := genesisState.GetAccount(common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"))
	assert.NoError(t, err)

	assert.Equal(t, len(acc.Writers()), 1)
	assert.Equal(t, acc.Writers(), [][]byte{byteutils.Hex2Bytes(writer)})

	genesisState.BeginBatch()
	assert.NoError(t, genesisState.AcceptTransaction(tx, test.GenesisBlock.Timestamp()))
	genesisState.Commit()

	removePayload := core.NewRemoveWriterPayload(common.HexToAddress(writer))
	removePayloadBuf, err := removePayload.ToBytes()
	assert.NoError(t, err)
	txRemove, err := core.NewTransaction(test.ChainID,
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
	txRegister, err := core.NewTransaction(test.ChainID,
		common.HexToAddress(owner),
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationRegisterWKey, payloadBuf)

	txDelegated, err := core.NewTransaction(test.ChainID,
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

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	assert.Equal(t, core.ErrInvalidTxDelegation, txDelegated.VerifyDelegation(genesisState))

	genesisState.BeginBatch()
	assert.NoError(t, txRegister.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txRegister, test.GenesisBlock.Timestamp()))
	genesisState.Commit()

	assert.NoError(t, txDelegated.VerifyDelegation(genesisState))
}

func TestAddRecord(t *testing.T) {
	recordHash := common.HexToHash("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	storage := "ipfs"
	encKey := byteutils.Hex2Bytes("abcdef")
	seed := byteutils.Hex2Bytes("5eed")
	payload := core.NewAddRecordPayload(recordHash, storage, encKey, seed)
	payloadBuf, err := payload.ToBytes()
	assert.NoError(t, err)
	owner := common.HexToAddress("02bdc97dfc02502c5b8301ff46cbbb0dce56cd96b0af75edc50560630de5b0a472")
	txAddRecord, err := core.NewTransaction(test.ChainID,
		owner,
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationAddRecord, payloadBuf)
	assert.NoError(t, err)

	privKey, err := secp256k1.NewPrivateKeyFromHex("0e2c9d389320398e7f09de9764e3892c6a9cc9b86b4bb1d64abc1bd995e363e7")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, txAddRecord.SignThis(sig))

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, txAddRecord.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txAddRecord, test.GenesisBlock.Timestamp()))
	genesisState.Commit()

	record, err := genesisState.GetRecord(recordHash)
	assert.NoError(t, err)
	assert.Equal(t, record.Hash, recordHash.Bytes())
	assert.Equal(t, record.Storage, storage)
	assert.Equal(t, len(record.Readers), 1)
	assert.Equal(t, record.Readers[0].Address, owner.Bytes())
	assert.Equal(t, record.Readers[0].EncKey, encKey)
	assert.Equal(t, record.Readers[0].Seed, seed)
}

func TestAddRecordReader(t *testing.T) {
	recordHash := common.HexToHash("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	storage := "ipfs"
	ownerEncKey := byteutils.Hex2Bytes("abcdef")
	ownerSeed := byteutils.Hex2Bytes("5eed")
	addRecordPayload := core.NewAddRecordPayload(recordHash, storage, ownerEncKey, ownerSeed)
	addRecordPayloadBuf, err := addRecordPayload.ToBytes()
	assert.NoError(t, err)
	owner := common.HexToAddress("02bdc97dfc02502c5b8301ff46cbbb0dce56cd96b0af75edc50560630de5b0a472")
	txAddRecord, err := core.NewTransaction(test.ChainID,
		owner,
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationAddRecord, addRecordPayloadBuf)
	assert.NoError(t, err)

	privKey, err := secp256k1.NewPrivateKeyFromHex("0e2c9d389320398e7f09de9764e3892c6a9cc9b86b4bb1d64abc1bd995e363e7")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, txAddRecord.SignThis(sig))

	reader := common.HexToAddress("03c5e1fa1ee82af7398ae8cc10ae12dc0ee9692cb06346810e3af74cbd3811276f")
	readerEncKey := byteutils.Hex2Bytes("123456")
	readerSeed := byteutils.Hex2Bytes("2eed")
	addRecordReaderPayload := core.NewAddRecordReaderPayload(recordHash, reader, readerEncKey, readerSeed)
	addRecordReaderPayloadBuf, err := addRecordReaderPayload.ToBytes()
	assert.NoError(t, err)

	txAddRecordReader, err := core.NewTransaction(test.ChainID,
		owner,
		common.Address{},
		util.Uint128Zero(), 2,
		core.TxOperationAddRecordReader, addRecordReaderPayloadBuf)
	assert.NoError(t, err)
	assert.NoError(t, txAddRecordReader.SignThis(sig))

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, txAddRecord.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txAddRecord, test.GenesisBlock.Timestamp()))
	assert.NoError(t, txAddRecordReader.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txAddRecordReader, test.GenesisBlock.Timestamp()))
	genesisState.Commit()

	record, err := genesisState.GetRecord(recordHash)
	assert.NoError(t, err)
	assert.Equal(t, record.Hash, recordHash.Bytes())
	assert.Equal(t, record.Storage, storage)
	assert.Equal(t, len(record.Readers), 2)
	assert.Equal(t, record.Readers[0].Address, owner.Bytes())
	assert.Equal(t, record.Readers[0].EncKey, ownerEncKey)
	assert.Equal(t, record.Readers[0].Seed, ownerSeed)
	assert.Equal(t, record.Readers[1].Address, reader.Bytes())
	assert.Equal(t, record.Readers[1].EncKey, readerEncKey)
	assert.Equal(t, record.Readers[1].Seed, readerSeed)

}
