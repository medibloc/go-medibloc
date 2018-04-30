package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestTransaction_VerifyIntegrity(t *testing.T) {
	testCount := 3
	type keyPair struct {
		pubKey []byte
		priKey string
	}

	keyPairs := []keyPair{
		{
			byteutils.Hex2Bytes("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c"),
			"ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea",
		},
		{
			byteutils.Hex2Bytes("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
			"bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0",
		},
		{
			byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e21"),
			"b108356a113edaaf537b6cd4f506f72787d69de3c3465adc30741653949e2173",
		},
	}

	type testTx struct {
		name    string
		tx      *core.Transaction
		privKey signature.PrivateKey
		count   int
	}

	var tests []testTx
	ks := keystore.NewKeyStore()
	for index := 0; index < testCount; index++ {

		from := common.BytesToAddress(keyPairs[index].pubKey)
		to := test.MockAddress(t, ks)

		tx, err := core.NewTransaction(test.ChainID, from, to, util.Uint128Zero(), 1, core.TxPayloadBinaryType, []byte("datadata"))
		assert.NoError(t, err)

		sig, err := crypto.NewSignature(algorithm.SECP256K1)
		assert.NoError(t, err)
		key, err := secp256k1.NewPrivateKeyFromHex(keyPairs[index].priKey)
		assert.NoError(t, err)
		sig.InitSign(key)
		assert.NoError(t, tx.SignThis(sig))
		tests = append(tests, testTx{string(index), tx, key, 1})
	}
	for _, tt := range tests {
		for index := 0; index < tt.count; index++ {
			t.Run(tt.name, func(t *testing.T) {
				sig, err := crypto.NewSignature(algorithm.SECP256K1)
				assert.NoError(t, err)
				signature.InitSign(tt.privKey)
				err = tt.tx.SignThis(signature)
				assert.NoErrorf(t, err, "Sign() error = %v", err)
				err = tt.tx.VerifyIntegrity(test.ChainID)
				assert.NoErrorf(t, err, "verify failed:%s", err)
			})
		}
	}
}

func TestRegisterWriteKey(t *testing.T) {
	genesis, dynasties := test.NewTestGenesisBlock(t)
	writer := dynasties[0].Addr
	payload := core.NewRegisterWriterPayload(writer)
	payloadBuf, err := payload.ToBytes()
	assert.NoError(t, err)
	tx, err := core.NewTransaction(test.ChainID,
		dynasties[1].Addr,
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationRegisterWKey, payloadBuf)

	privKey := dynasties[1].PrivKey
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, tx.SignThis(sig))

	genesisState, err := genesis.State().Clone()
	assert.NoError(t, err)
	genesisState.BeginBatch()
	assert.NoError(t, tx.ExecuteOnState(genesisState))
	genesisState.Commit()
	genesisState.BeginBatch()
	assert.Equal(t, tx.ExecuteOnState(genesisState), core.ErrWriterAlreadyRegistered)

	acc, err := genesisState.GetAccount(dynasties[1].Addr)
	assert.NoError(t, err)

	assert.Equal(t, len(acc.Writers()), 1)
	assert.Equal(t, acc.Writers(), [][]byte{writer.Bytes()})

	genesisState.BeginBatch()
	assert.NoError(t, genesisState.AcceptTransaction(tx, genesis.Timestamp()))
	genesisState.Commit()

	removePayload := core.NewRemoveWriterPayload(writer)
	removePayloadBuf, err := removePayload.ToBytes()
	assert.NoError(t, err)
	txRemove, err := core.NewTransaction(test.ChainID,
		dynasties[1].Addr,
		common.Address{},
		util.Uint128Zero(), 2,
		core.TxOperationRemoveWKey, removePayloadBuf)
	assert.NoError(t, txRemove.SignThis(sig))

	genesisState.BeginBatch()
	assert.NoError(t, txRemove.ExecuteOnState(genesisState))
	genesisState.Commit()

	acc, err = genesisState.GetAccount(dynasties[1].Addr)
	assert.NoError(t, err)

	assert.Equal(t, len(acc.Writers()), 0)
	genesisState.BeginBatch()
	assert.Equal(t, txRemove.ExecuteOnState(genesisState), core.ErrWriterNotFound)
}

func TestVerifyDelegation(t *testing.T) {
	genesis, dynasties := test.NewTestGenesisBlock(t)

	writer := dynasties[0]
	payload := core.NewRegisterWriterPayload(writer.Addr)
	payloadBuf, err := payload.ToBytes()
	assert.NoError(t, err)
	owner := dynasties[1]
	txRegister, err := core.NewTransaction(test.ChainID,
		owner.Addr,
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationRegisterWKey, payloadBuf)

	txDelegated, err := core.NewTransaction(test.ChainID,
		owner.Addr,
		dynasties[2].Addr,
		util.NewUint128FromUint(10), 2,
		core.TxPayloadBinaryType, []byte{})
	assert.NoError(t, err)

	writerPrivKey := writer.PrivKey
	assert.NoError(t, err)
	sigW, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sigW.InitSign(writerPrivKey)
	assert.NoError(t, txDelegated.SignThis(sigW))

	privKey := owner.PrivKey
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, txRegister.SignThis(sig))

	genesisState, err := genesis.State().Clone()
	assert.NoError(t, err)

	assert.Equal(t, core.ErrInvalidTxDelegation, txDelegated.VerifyDelegation(genesisState))

	genesisState.BeginBatch()
	assert.NoError(t, txRegister.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txRegister, genesis.Timestamp()))
	genesisState.Commit()

	assert.NoError(t, txDelegated.VerifyDelegation(genesisState))
}

func TestAddRecord(t *testing.T) {
	genesis, dynasties := test.NewTestGenesisBlock(t)

	recordHash := common.HexToHash("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	storage := "ipfs"
	encKey := []byte("abcdef")
	seed := []byte("5eed")
	payload := core.NewAddRecordPayload(recordHash, storage, encKey, seed)
	payloadBuf, err := payload.ToBytes()
	assert.NoError(t, err)
	owner := dynasties[0]
	txAddRecord, err := core.NewTransaction(test.ChainID,
		owner.Addr,
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationAddRecord, payloadBuf)
	assert.NoError(t, err)

	privKey := owner.PrivKey
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, txAddRecord.SignThis(sig))

	genesisState, err := genesis.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, txAddRecord.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txAddRecord, genesis.Timestamp()))
	genesisState.Commit()

	record, err := genesisState.GetRecord(recordHash)
	assert.NoError(t, err)
	assert.Equal(t, record.Hash, recordHash.Bytes())
	assert.Equal(t, record.Storage, storage)
	assert.Equal(t, len(record.Readers), 1)
	assert.Equal(t, record.Readers[0].Address, owner.Addr.Bytes())
	assert.Equal(t, record.Readers[0].EncKey, encKey)
	assert.Equal(t, record.Readers[0].Seed, seed)
}

func TestAddRecordReader(t *testing.T) {
	genesis, dynasties := test.NewTestGenesisBlock(t)

	recordHash := common.HexToHash("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	storage := "ipfs"
	ownerEncKey := []byte("abcdef")
	ownerSeed := []byte("5eed")
	addRecordPayload := core.NewAddRecordPayload(recordHash, storage, ownerEncKey, ownerSeed)
	addRecordPayloadBuf, err := addRecordPayload.ToBytes()
	assert.NoError(t, err)
	owner := dynasties[0]
	txAddRecord, err := core.NewTransaction(test.ChainID,
		owner.Addr,
		common.Address{},
		util.Uint128Zero(), 1,
		core.TxOperationAddRecord, addRecordPayloadBuf)
	assert.NoError(t, err)

	privKey := owner.PrivKey
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, txAddRecord.SignThis(sig))

	reader := dynasties[1]
	readerEncKey := []byte("123456")
	readerSeed := []byte("2eed")
	addRecordReaderPayload := core.NewAddRecordReaderPayload(recordHash, reader.Addr, readerEncKey, readerSeed)
	addRecordReaderPayloadBuf, err := addRecordReaderPayload.ToBytes()
	assert.NoError(t, err)

	txAddRecordReader, err := core.NewTransaction(test.ChainID,
		owner.Addr,
		common.Address{},
		util.Uint128Zero(), 2,
		core.TxOperationAddRecordReader, addRecordReaderPayloadBuf)
	assert.NoError(t, err)
	assert.NoError(t, txAddRecordReader.SignThis(sig))

	genesisState, err := genesis.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, txAddRecord.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txAddRecord, genesis.Timestamp()))
	assert.NoError(t, txAddRecordReader.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(txAddRecordReader, genesis.Timestamp()))
	genesisState.Commit()

	record, err := genesisState.GetRecord(recordHash)
	assert.NoError(t, err)
	assert.Equal(t, record.Hash, recordHash.Bytes())
	assert.Equal(t, record.Storage, storage)
	assert.Equal(t, len(record.Readers), 2)
	assert.Equal(t, record.Readers[0].Address, owner.Addr.Bytes())
	assert.Equal(t, record.Readers[0].EncKey, ownerEncKey)
	assert.Equal(t, record.Readers[0].Seed, ownerSeed)
	assert.Equal(t, record.Readers[1].Address, reader.Addr.Bytes())
	assert.Equal(t, record.Readers[1].EncKey, readerEncKey)
	assert.Equal(t, record.Readers[1].Seed, readerSeed)
}

func TestVest(t *testing.T) {
	genesis, dynasties := test.NewTestGenesisBlock(t)

	from := dynasties[0]
	tx, err := core.NewTransaction(
		test.ChainID,
		from.Addr,
		common.Address{},
		util.NewUint128FromUint(333), 1,
		core.TxOperationVest, []byte{},
	)
	assert.NoError(t, err)
	privKey := from.PrivKey
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, tx.SignThis(sig))

	genesisState, err := genesis.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, tx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(tx, genesis.Timestamp()))
	genesisState.Commit()

	acc, err := genesisState.GetAccount(from.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(333)))
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-333)))
}

func TestWithdrawVesting(t *testing.T) {
	genesis, dynasties := test.NewTestGenesisBlock(t)

	from := dynasties[0]
	vestTx, err := core.NewTransaction(
		test.ChainID,
		from.Addr,
		common.Address{},
		util.NewUint128FromUint(333), 1,
		core.TxOperationVest, []byte{},
	)
	withdrawTx, err := core.NewTransaction(
		test.ChainID,
		from.Addr,
		common.Address{},
		util.NewUint128FromUint(333), 2,
		core.TxOperationWithdrawVesting, []byte{})
	withdrawTx.SetTimestamp(int64(0))
	assert.NoError(t, err)
	privKey := from.PrivKey
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, vestTx.SignThis(sig))
	assert.NoError(t, withdrawTx.SignThis(sig))

	genesisState, err := genesis.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, vestTx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(vestTx, genesis.Timestamp()))
	assert.NoError(t, withdrawTx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(withdrawTx, genesis.Timestamp()))
	genesisState.Commit()

	acc, err := genesisState.GetAccount(from.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(333)))
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-333)))
	tasks := genesisState.GetReservedTasks()
	assert.Equal(t, 3, len(tasks))
	for i := 0; i < len(tasks); i++ {
		assert.Equal(t, core.RtWithdrawType, tasks[i].TaskType())
		assert.Equal(t, from.Addr, tasks[i].From())
		assert.Equal(t, withdrawTx.Timestamp()+int64(i+1)*core.RtWithdrawInterval, tasks[i].Timestamp())
	}
}

func TestBecomeCandidate(t *testing.T) {
	from := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	tx, err := core.NewTransaction(
		test.ChainID,
		from,
		common.Address{},
		util.NewUint128FromUint(10), 1,
		core.TxOperationBecomeCandidate, []byte{},
	)
	assert.NoError(t, err)
	privKey, err := secp256k1.NewPrivateKeyFromHex("ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, tx.SignThis(sig))

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, tx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(tx, test.GenesisBlock.Timestamp()))
	genesisState.Commit()

	acc, err := genesisState.GetAccount(from)
	assert.NoError(t, err)
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-10)))
	candidate, err := genesisState.GetCandidate(from)
	assert.NoError(t, err)
	tenBytes, err := util.NewUint128FromUint(10).ToFixedSizeByteSlice()
	assert.NoError(t, err)
	assert.Equal(t, candidate.Collateral, tenBytes)
}

func TestBecomeCandidateAlreadyCandidate(t *testing.T) {
	from := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	tx1, err := core.NewTransaction(
		test.ChainID,
		from,
		common.Address{},
		util.NewUint128FromUint(10), 1,
		core.TxOperationBecomeCandidate, []byte{},
	)
	tx2, err := core.NewTransaction(
		test.ChainID,
		from,
		common.Address{},
		util.NewUint128FromUint(10), 2,
		core.TxOperationBecomeCandidate, []byte{},
	)
	assert.NoError(t, err)
	privKey, err := secp256k1.NewPrivateKeyFromHex("ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, tx1.SignThis(sig))
	assert.NoError(t, tx2.SignThis(sig))

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, tx1.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(tx1, test.GenesisBlock.Timestamp()))
	assert.Equal(t, core.ErrAlreadyInCandidacy, tx2.ExecuteOnState(genesisState))
}

func TestBecomeCandidateTooMuchCollateral(t *testing.T) {
	from := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	tx, err := core.NewTransaction(
		test.ChainID,
		from,
		common.Address{},
		util.NewUint128FromUint(1000000001), 1,
		core.TxOperationBecomeCandidate, []byte{},
	)
	assert.NoError(t, err)
	privKey, err := secp256k1.NewPrivateKeyFromHex("ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, tx.SignThis(sig))

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.Equal(t, core.ErrBalanceNotEnough, tx.ExecuteOnState(genesisState))
}

func TestQuitCandidacy(t *testing.T) {
	from := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	becomeTx, err := core.NewTransaction(
		test.ChainID,
		from,
		common.Address{},
		util.NewUint128FromUint(10), 1,
		core.TxOperationBecomeCandidate, []byte{},
	)
	quitTx, err := core.NewTransaction(
		test.ChainID,
		from,
		common.Address{},
		util.NewUint128FromUint(0), 2,
		core.TxOperationQuitCandidacy, []byte{},
	)
	assert.NoError(t, err)
	privKey, err := secp256k1.NewPrivateKeyFromHex("ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea")
	assert.NoError(t, err)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(privKey)
	assert.NoError(t, becomeTx.SignThis(sig))
	assert.NoError(t, quitTx.SignThis(sig))

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, becomeTx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(becomeTx, test.GenesisBlock.Timestamp()))
	assert.NoError(t, quitTx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(quitTx, test.GenesisBlock.Timestamp()))
	genesisState.Commit()

	acc, err := genesisState.GetAccount(from)
	assert.NoError(t, err)
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000)))
	_, err = genesisState.GetCandidate(from)
	assert.Equal(t, core.ErrNotFound, err)
}

func TestVote(t *testing.T) {
	voter := common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	voted := common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939")
	becomeTx, err := core.NewTransaction(
		test.ChainID,
		voted,
		common.Address{},
		util.NewUint128FromUint(10), 1,
		core.TxOperationBecomeCandidate, []byte{},
	)
	assert.NoError(t, err)
	voteTx, err := core.NewTransaction(
		test.ChainID,
		voter,
		voted,
		util.NewUint128FromUint(0), 1,
		core.TxOperationVote, []byte{},
	)
	assert.NoError(t, err)

	votedPrivKey, err := secp256k1.NewPrivateKeyFromHex("bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0")
	assert.NoError(t, err)
	votedSig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	votedSig.InitSign(votedPrivKey)
	assert.NoError(t, becomeTx.SignThis(votedSig))

	voterPrivKey, err := secp256k1.NewPrivateKeyFromHex("ee8ea71e9501306fdd00c6e58b2ede51ca125a583858947ff8e309abf11d37ea")
	assert.NoError(t, err)
	voterSig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	voterSig.InitSign(voterPrivKey)
	assert.NoError(t, voteTx.SignThis(voterSig))

	genesisState, err := test.GenesisBlock.State().Clone()
	assert.NoError(t, err)

	genesisState.BeginBatch()
	assert.NoError(t, becomeTx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(becomeTx, test.GenesisBlock.Timestamp()))
	assert.NoError(t, voteTx.ExecuteOnState(genesisState))
	assert.NoError(t, genesisState.AcceptTransaction(voteTx, test.GenesisBlock.Timestamp()))
	genesisState.Commit()

	voterAcc, err := genesisState.GetAccount(voter)
	assert.NoError(t, err)
	assert.Equal(t, voted.Bytes(), voterAcc.Voted())
}
