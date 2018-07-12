// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package core_test

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/txutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		to := testutil.MockAddress(t, ks)

		tx, err := core.NewTransaction(testutil.ChainID, from, to, util.Uint128Zero(), 1, core.TxPayloadBinaryType, []byte("datadata"))
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
				signature, err := crypto.NewSignature(algorithm.SECP256K1)
				assert.NoError(t, err)
				signature.InitSign(tt.privKey)
				err = tt.tx.SignThis(signature)
				assert.NoErrorf(t, err, "Sign() error = %v", err)
				err = tt.tx.VerifyIntegrity(testutil.ChainID)
				assert.NoErrorf(t, err, "verify failed:%s", err)
			})
		}
	}
}

func TestAddRecord(t *testing.T) {
	genesis, dynasties, _ := testutil.NewTestGenesisBlock(t, 21)

	recordHash := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	payload := core.NewAddRecordPayload(recordHash)
	owner := dynasties[0]

	transaction := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(owner.Addr).SetNonce(1).
		SetType(core.TxOperationAddRecord).SetPayload(payload).CalcHash().Sign(owner.PrivKey).Build()

	newTxFunc := testutil.TxMap[transaction.Type()]
	tx, err := newTxFunc(transaction)
	require.NoError(t, err)

	genesis.BeginBatch()
	require.NoError(t, tx.Execute(genesis))
	require.NoError(t, genesis.State().AcceptTransaction(transaction, genesis.Timestamp()))
	genesis.Commit()

	record, err := genesis.State().GetRecord(recordHash)
	assert.NoError(t, err)
	assert.Equal(t, record.Hash, recordHash)
}

func TestVest(t *testing.T) {
	genesis, dynasties, _ := testutil.NewTestGenesisBlock(t, 21)

	from := dynasties[0]

	transaction := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(from.Addr).SetNonce(1).
		SetValue(333).SetNonce(1).SetType(core.TxOperationVest).CalcHash().Sign(from.PrivKey).Build()

	newTxFunc := testutil.TxMap[transaction.Type()]
	tx, err := newTxFunc(transaction)
	require.NoError(t, err)

	genesis.BeginBatch()
	require.NoError(t, tx.Execute(genesis))
	require.NoError(t, genesis.State().AcceptTransaction(transaction, genesis.Timestamp()))
	genesis.Commit()

	acc, err := genesis.State().GetAccount(from.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(333)))
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-333)))
}

func TestWithdrawVesting(t *testing.T) {
	genesis, dynasties, _ := testutil.NewTestGenesisBlock(t, 21)

	from := dynasties[0]
	vestTx, err := core.NewTransaction(
		testutil.ChainID,
		from.Addr,
		common.Address{},
		util.NewUint128FromUint(333), 1,
		core.TxOperationVest, []byte{},
	)
	withdrawTx, err := core.NewTransaction(
		testutil.ChainID,
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

func TestBecomeAndQuitCandidate(t *testing.T) {
	genesis, _, distributed := testutil.NewTestGenesisBlock(t, testutil.DynastySize)
	candidate := distributed[testutil.DynastySize]

	tb := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(candidate.Addr).SetNonce(1).
		SetType(dpos.TxOperationBecomeCandidate).CalcHash().Sign(candidate.PrivKey)

	tb.SetValue(1000000001)
	transaction1 := tb.Build()

	tb.SetValue(10)
	transaction2 := tb.Build()

	tb.SetNonce(2)
	transaction3 := tb.Build()

	newTxFunc := testutil.TxMap[transaction1.Type()]
	tx1, err := newTxFunc(transaction1)
	require.NoError(t, err)
	tx2, err := newTxFunc(transaction2)
	require.NoError(t, err)
	tx3, err := newTxFunc(transaction3)
	require.NoError(t, err)

	genesis.State().BeginBatch()
	assert.Equal(t, core.ErrBalanceNotEnough, tx1.Execute(genesis))
	assert.NoError(t, tx2.Execute(genesis))
	assert.Equal(t, dpos.ErrAlreadyInCandidacy, tx3.Execute(genesis))
	assert.NoError(t, genesis.State().AcceptTransaction(transaction2, genesis.Timestamp()))
	genesis.State().Commit()

	acc, err := genesis.State().GetAccount(candidate.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-10)))
	candidateBytes, err := genesis.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.NoError(t, err)

	candidatePb := new(dpospb.Candidate)
	proto.Unmarshal(candidateBytes, candidatePb)

	tenBytes, err := util.NewUint128FromUint(10).ToFixedSizeByteSlice()
	assert.NoError(t, err)
	assert.Equal(t, candidatePb.Collatral, tenBytes)

	tbQuit := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(candidate.Addr).SetNonce(2).
		SetType(dpos.TxOperationQuitCandidacy)

	transactionQuit := tbQuit.CalcHash().Sign(candidate.PrivKey).Build()

	newTxFuncQuit := testutil.TxMap[transactionQuit.Type()]
	txQuit, err := newTxFuncQuit(transactionQuit)
	require.NoError(t, err)

	genesis.State().BeginBatch()
	assert.NoError(t, txQuit.Execute(genesis))
	assert.NoError(t, genesis.State().AcceptTransaction(transaction2, genesis.Timestamp()))
	genesis.State().Commit()

	acc, err = genesis.State().GetAccount(candidate.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000)))
	_, err = genesis.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.Equal(t, trie.ErrNotFound, err)

}

func TestVote(t *testing.T) {
	genesis, _, distributed := testutil.NewTestGenesisBlock(t, testutil.DynastySize)
	candidate := distributed[testutil.DynastySize]
	voter := distributed[testutil.DynastySize+1]

	transactionVest := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(voter.Addr).SetNonce(1).
		SetValue(333).SetNonce(1).SetType(core.TxOperationVest).CalcHash().Sign(voter.PrivKey).Build()

	newTxFunc := testutil.TxMap[transactionVest.Type()]
	txVest, err := newTxFunc(transactionVest)
	require.NoError(t, err)

	transactionBecome := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(candidate.Addr).SetNonce(1).
		SetType(dpos.TxOperationBecomeCandidate).SetValue(10).CalcHash().Sign(candidate.PrivKey).Build()

	newTxFunc = testutil.TxMap[transactionBecome.Type()]
	txBecome, err := newTxFunc(transactionBecome)
	require.NoError(t, err)

	transactionVote := txutil.NewTransactionBuilder(t).SetChainID(testutil.ChainID).SetFrom(voter.Addr).
		SetNonce(2).SetTo(candidate.Addr).SetType(dpos.TxOperationVote).CalcHash().Sign(voter.PrivKey).Build()

	newTxFunc = testutil.TxMap[transactionVote.Type()]
	txVote, err := newTxFunc(transactionVote)
	require.NoError(t, err)

	genesis.State().BeginBatch()
	assert.NoError(t, txBecome.Execute(genesis))
	assert.NoError(t, genesis.State().AcceptTransaction(transactionBecome, genesis.Timestamp()))

	assert.NoError(t, txVest.Execute(genesis))
	assert.NoError(t, genesis.State().AcceptTransaction(transactionVest, genesis.Timestamp()))

	assert.NoError(t, txVote.Execute(genesis))
	assert.NoError(t, genesis.State().AcceptTransaction(transactionVote, genesis.Timestamp()))

	genesis.State().Commit()

	acc, err := genesis.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(333)))

	candidateBytes, err := genesis.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.NoError(t, err)

	candidatePb := new(dpospb.Candidate)
	proto.Unmarshal(candidateBytes, candidatePb)

	expectedVotePower, err := util.NewUint128FromUint(333).ToFixedSizeByteSlice()
	require.NoError(t, err)
	assert.Equal(t, candidatePb.VotePower, expectedVotePower)

}

func TestAddCertification(t *testing.T) {
	genesis, _, users := testutil.NewTestGenesisBlock(t, 21)

	certs := []struct {
		issuer         common.Address
		issuerPrivKey  signature.PrivateKey
		certified      common.Address
		issueTime      int64
		expirationTime int64
		hash           []byte
	}{
		{
			users[0].Addr,
			users[0].PrivKey,
			users[1].Addr,
			time.Now().Unix(),
			time.Now().Unix() + int64(100000),
			byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e"),
		},
	}

	st := genesis.State()

	payload := core.NewAddCertificationPayload(certs[0].issueTime, certs[0].expirationTime, certs[0].hash)
	payloadBuf, err := payload.ToBytes()
	assert.NoError(t, err)

	addCertTx, err := core.NewTransaction(testutil.ChainID, certs[0].issuer, certs[0].certified,
		util.Uint128Zero(), 1, core.TxOperationAddCertification, payloadBuf)
	assert.NoError(t, err)

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(certs[0].issuerPrivKey)
	assert.NoError(t, addCertTx.SignThis(sig))

	st.BeginBatch()
	assert.NoError(t, addCertTx.ExecuteOnState(st))
	assert.NoError(t, st.AcceptTransaction(addCertTx, genesis.Timestamp()))
	st.Commit()

	accs := []core.Account{}
	for i := 0; i < 2; i++ {
		acc, err := st.GetAccount(users[i].Addr)
		assert.NoError(t, err)
		accs = append(accs, acc)
	}
	assert.Equal(t, 1, len(accs[0].CertsIssued()))
	assert.Equal(t, 0, len(accs[0].CertsReceived()))
	assert.Equal(t, 0, len(accs[1].CertsIssued()))
	assert.Equal(t, 1, len(accs[1].CertsReceived()))

	assert.Equal(t, byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e"), accs[0].CertsIssued()[0])
	assert.Equal(t, byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e"), accs[1].CertsReceived()[0])

	certHash := accs[0].CertsIssued()[0]

	cert, err := st.GetCertification(certs[0].hash)
	assert.NoError(t, err)

	assert.Equal(t, cert.CertificateHash, certHash)
	assert.Equal(t, cert.Issuer, users[0].Addr.Bytes())
	assert.Equal(t, cert.Certified, users[1].Addr.Bytes())
	assert.Equal(t, cert.IssueTime, certs[0].issueTime)
	assert.Equal(t, cert.ExpirationTime, certs[0].expirationTime)
	assert.Equal(t, cert.RevocationTime, int64(0))
}

func TestRevokeCertification(t *testing.T) {
	genesis, _, users := testutil.NewTestGenesisBlock(t, 21)

	certs := []struct {
		issuer         common.Address
		issuerPrivKey  signature.PrivateKey
		certified      common.Address
		issueTime      int64
		expirationTime int64
		revocationTime int64
		hash           []byte
	}{
		{
			users[0].Addr,
			users[0].PrivKey,
			users[1].Addr,
			time.Now().Unix(),
			time.Now().Unix() + int64(100000),
			time.Now().Unix() + int64(100),
			byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e"),
		},
	}

	st := genesis.State()

	addCertPayload := core.NewAddCertificationPayload(certs[0].issueTime, certs[0].expirationTime, certs[0].hash)
	addCertPayloadBuf, err := addCertPayload.ToBytes()
	assert.NoError(t, err)

	addCertTx, err := core.NewTransaction(testutil.ChainID, certs[0].issuer, certs[0].certified,
		util.Uint128Zero(), 1, core.TxOperationAddCertification, addCertPayloadBuf)
	assert.NoError(t, err)

	revokeCertPayload := core.NewRevokeCertificationPayload(certs[0].hash)
	revokeCertPayloadBuf, err := revokeCertPayload.ToBytes()
	assert.NoError(t, err)

	revokeCertTx, err := core.NewTransaction(testutil.ChainID, certs[0].issuer, common.Address{},
		util.Uint128Zero(), 2, core.TxOperationRevokeCertification, revokeCertPayloadBuf)

	revokeCertTx.SetTimestamp(certs[0].revocationTime)

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(certs[0].issuerPrivKey)
	assert.NoError(t, addCertTx.SignThis(sig))
	assert.NoError(t, revokeCertTx.SignThis(sig))

	st.BeginBatch()
	assert.NoError(t, addCertTx.ExecuteOnState(st))
	assert.NoError(t, st.AcceptTransaction(addCertTx, genesis.Timestamp()))
	assert.NoError(t, revokeCertTx.ExecuteOnState(st))
	assert.NoError(t, st.AcceptTransaction(revokeCertTx, genesis.Timestamp()))
	st.Commit()

	accs := []core.Account{}
	for i := 0; i < 2; i++ {
		acc, err := st.GetAccount(users[i].Addr)
		assert.NoError(t, err)
		accs = append(accs, acc)
	}
	assert.Equal(t, 1, len(accs[0].CertsIssued()))
	assert.Equal(t, 0, len(accs[0].CertsReceived()))
	assert.Equal(t, 0, len(accs[1].CertsIssued()))
	assert.Equal(t, 1, len(accs[1].CertsReceived()))

	certHash := accs[0].CertsIssued()[0]

	cert, err := st.GetCertification(certs[0].hash)
	assert.NoError(t, err)

	assert.Equal(t, cert.CertificateHash, certHash)
	assert.Equal(t, cert.Issuer, users[0].Addr.Bytes())
	assert.Equal(t, cert.Certified, users[1].Addr.Bytes())
	assert.Equal(t, cert.IssueTime, certs[0].issueTime)
	assert.Equal(t, cert.ExpirationTime, certs[0].expirationTime)
	assert.Equal(t, cert.RevocationTime, certs[0].revocationTime)
}

func TestRevokeCertificationByInvalidAccount(t *testing.T) {
	genesis, _, users := testutil.NewTestGenesisBlock(t, 21)

	certs := []struct {
		issuer         common.Address
		issuerPrivKey  signature.PrivateKey
		revoker        common.Address
		revokerPrivKey signature.PrivateKey
		certified      common.Address
		issueTime      int64
		expirationTime int64
		revocationTime int64
		hash           []byte
	}{
		{
			users[0].Addr,
			users[0].PrivKey,
			users[2].Addr,
			users[2].PrivKey,
			users[1].Addr,
			time.Now().Unix(),
			time.Now().Unix() + int64(100000),
			time.Now().Unix() + int64(100),
			byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e"),
		},
	}

	st := genesis.State()

	addCertPayload := core.NewAddCertificationPayload(certs[0].issueTime, certs[0].expirationTime, certs[0].hash)
	addCertPayloadBuf, err := addCertPayload.ToBytes()
	assert.NoError(t, err)

	addCertTx, err := core.NewTransaction(testutil.ChainID, certs[0].issuer, certs[0].certified,
		util.Uint128Zero(), 1, core.TxOperationAddCertification, addCertPayloadBuf)
	assert.NoError(t, err)

	revokeCertPayload := core.NewRevokeCertificationPayload(certs[0].hash)
	revokeCertPayloadBuf, err := revokeCertPayload.ToBytes()
	assert.NoError(t, err)

	revokeCertTx, err := core.NewTransaction(testutil.ChainID, certs[0].revoker, common.Address{},
		util.Uint128Zero(), 2, core.TxOperationRevokeCertification, revokeCertPayloadBuf)

	revokeCertTx.SetTimestamp(certs[0].revocationTime)

	addSig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	addSig.InitSign(certs[0].issuerPrivKey)
	assert.NoError(t, addCertTx.SignThis(addSig))

	revokeSig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	revokeSig.InitSign(certs[0].revokerPrivKey)
	assert.NoError(t, revokeCertTx.SignThis(revokeSig))

	st.BeginBatch()
	assert.NoError(t, addCertTx.ExecuteOnState(st))
	assert.NoError(t, st.AcceptTransaction(addCertTx, genesis.Timestamp()))
	assert.Error(t, core.ErrInvalidCertificationRevoker, revokeCertTx.ExecuteOnState(st))
}
