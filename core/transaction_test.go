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

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
)

//func TestTransaction_VerifyIntegrity(t *testing.T) {
//	testCount := 3
//	type keyPair struct {
//		pubKey []byte
//		priKey string
//	}
//
//	var keyPairs testutil.AddrKeyPairs
//
//	for i :=0; i<testCount; i++ {
//		keyPairs = append(keyPairs, testutil.NewAddrKeyPair(t))
//	}
//
//	type testTx struct {
//		name    string
//		tx      *core.Transaction
//		privKey signature.PrivateKey
//		count   int
//	}
//
//	var tests []testTx
//	for index := 0; index < testCount; index++ {
//
//		from := keyPairs[index].Addr
//		to := testutil.NewAddrKeyPair(t).Addr
//
//		tb := blockutil.New(t, testutil.DynastySize).
//			Tx().ChainID(testutil.ChainID).From(from).To(to).Value(0).
//			Nonce(1).Type(core.TxPayloadBinaryType)
//
//
//		sig, err := crypto.NewSignature(algorithm.SECP256K1)
//		assert.NoError(t, err)
//		key := keyPairs[index].PrivKey
//		sig.InitSign(key)
//		assert.NoError(t, tx.SignThis(sig))
//		tests = append(tests, testTx{string(index), tx, key, 1})
//	}
//	for _, tt := range tests {
//		for index := 0; index < tt.count; index++ {
//			t.Run(tt.name, func(t *testing.T) {
//				signature, err := crypto.NewSignature(algorithm.SECP256K1)
//				assert.NoError(t, err)
//				signature.InitSign(tt.privKey)
//				err = tt.tx.SignThis(signature)
//				assert.NoErrorf(t, err, "Sign() error = %v", err)
//				err = tt.tx.VerifyIntegrity(testutil.ChainID)
//				assert.NoErrorf(t, err, "verify failed:%s", err)
//			})
//		}
//	}
//}

func TestAddRecord(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	recordHash := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	payload := core.NewAddRecordPayload(recordHash)
	owner := bb.TokenDist[0]

	block := bb.
		Tx().Type(core.TxOpAddRecord).Payload(payload).Nonce(1).SignPair(owner).Execute().
		Build()

	record, err := block.State().GetRecord(recordHash)
	assert.NoError(t, err)
	assert.Equal(t, record.Hash, recordHash)
}

func TestVest(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	from := bb.TokenDist[0]

	bb = bb.
		Tx().Type(core.TxOpVest).Value(333).Nonce(1).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, uint64(1000000000-333)).
		Vesting(from.Addr, uint64(333))
}

//func TestWithdrawVesting(t *testing.T) {
//	genesis, dynasties, _ := testutil.NewTestGenesisBlock(t, 21)
//
//	from := dynasties[0]
//	vestTx, err := core.NewTransaction(
//		testutil.ChainID,
//		from.Addr,
//		common.Address{},
//		util.NewUint128FromUint(333), 1,
//		core.TxOpVest, []byte{},
//	)
//	withdrawTx, err := core.NewTransaction(
//		testutil.ChainID,
//		from.Addr,
//		common.Address{},
//		util.NewUint128FromUint(333), 2,
//		core.TxOpWithdrawVesting, []byte{})
//	withdrawTx.SetTimestamp(int64(0))
//	assert.NoError(t, err)
//	privKey := from.PrivKey
//	assert.NoError(t, err)
//	sig, err := crypto.NewSignature(algorithm.SECP256K1)
//	assert.NoError(t, err)
//	sig.InitSign(privKey)
//	assert.NoError(t, vestTx.SignThis(sig))
//	assert.NoError(t, withdrawTx.SignThis(sig))
//
//	genesisState, err := genesis.State().Clone()
//	assert.NoError(t, err)
//
//	genesisState.BeginBatch()
//	assert.NoError(t, vestTx.ExecuteOnState(genesisState))
//	assert.NoError(t, genesisState.AcceptTransaction(vestTx, genesis.Timestamp()))
//	assert.NoError(t, withdrawTx.ExecuteOnState(genesisState))
//	assert.NoError(t, genesisState.AcceptTransaction(withdrawTx, genesis.Timestamp()))
//	genesisState.Commit()
//
//	acc, err := genesisState.GetAccount(from.Addr)
//	assert.NoError(t, err)
//	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(333)))
//	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-333)))
//	tasks := genesisState.GetReservedTasks()
//	assert.Equal(t, 3, len(tasks))
//	for i := 0; i < len(tasks); i++ {
//		assert.Equal(t, core.RtWithdrawType, tasks[i].TaskType())
//		assert.Equal(t, from.Addr, tasks[i].From())
//		assert.Equal(t, withdrawTx.Timestamp()+int64(i+1)*core.RtWithdrawInterval, tasks[i].Timestamp())
//	}
//}

func TestAddAndRevokeCertification(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	issuer := bb.TokenDist[0]
	certified := bb.TokenDist[1]

	issueTime := time.Now().Unix()
	expirationTime := time.Now().Unix() + int64(100000)
	hash := byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")

	// Add certification Test
	addPayload := core.NewAddCertificationPayload(issueTime, expirationTime, hash)

	bb = bb.
		Tx().Nonce(1).Type(core.TxOpAddCertification).From(issuer.Addr).To(certified.Addr).Payload(addPayload).CalcHash().SignKey(issuer.PrivKey).Execute()

	block := bb.Build()

	issuerAcc, err := block.State().GetAccount(issuer.Addr)
	require.NoError(t, err)
	certifiedAcc, err := block.State().GetAccount(certified.Addr)
	require.NoError(t, err)

	assert.Equal(t, 1, len(issuerAcc.CertsIssued()))
	assert.Equal(t, 1, len(certifiedAcc.CertsReceived()))

	assert.Equal(t, hash, issuerAcc.CertsIssued()[0])
	assert.Equal(t, hash, certifiedAcc.CertsReceived()[0])

	cert, err := block.State().Certification(hash)
	assert.NoError(t, err)

	assert.Equal(t, cert.CertificateHash, hash)
	assert.Equal(t, cert.Issuer, issuer.Addr.Bytes())
	assert.Equal(t, cert.Certified, certified.Addr.Bytes())
	assert.Equal(t, cert.IssueTime, issueTime)
	assert.Equal(t, cert.ExpirationTime, expirationTime)
	assert.Equal(t, cert.RevocationTime, int64(0))
	t.Logf("Add certification test complete")

	// Revoke certification Test
	wrongRevoker := bb.TokenDist[2]
	revokeTime := time.Now().Unix() + int64(50000)
	revokePayload := core.NewRevokeCertificationPayload(hash)

	bb = bb.
		Tx().Nonce(1).Type(core.TxOpRevokeCertification).Payload(revokePayload).SignPair(wrongRevoker).ExecuteErr(core.ErrInvalidCertificationRevoker).
		Tx().Nonce(2).Type(core.TxOpRevokeCertification).Payload(revokePayload).Timestamp(expirationTime + int64(1)).SignPair(issuer).ExecuteErr(core.ErrCertAlreadyExpired).
		Tx().Nonce(2).Type(core.TxOpRevokeCertification).Payload(revokePayload).Timestamp(revokeTime).SignPair(issuer).Execute().
		Tx().Nonce(3).Type(core.TxOpRevokeCertification).Payload(revokePayload).Timestamp(revokeTime).SignPair(issuer).
		ExecuteErr(core.ErrCertAlreadyRevoked)
	block = bb.Build()

	issuerAcc, err = block.State().GetAccount(issuer.Addr)
	require.NoError(t, err)
	certifiedAcc, err = block.State().GetAccount(certified.Addr)
	require.NoError(t, err)

	cert, err = block.State().Certification(hash)
	require.NoError(t, err)

	assert.Equal(t, cert.RevocationTime, revokeTime)
	t.Logf("Revoke certification test complete")

}