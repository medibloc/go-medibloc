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
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[0]
	to := testutil.NewAddrKeyPair(t)

	bb.
		Tx().Type(core.TxOpSend).To(to.Addr).Value(1000000001).SignPair(from).ExecuteErr(core.ErrBalanceNotEnough).
		Tx().Type(core.TxOpSend).To(to.Addr).Value(10).SignPair(from).Execute().
		Expect().
		Balance(to.Addr, 10).
		Balance(from.Addr, 1000000000-10)

}

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

func TestVestAndWithdraw(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[0]
	vestingAmount := uint64(1000)
	withdrawAmount := uint64(301)

	bb = bb.
		Tx().Type(core.TxOpVest).Value(vestingAmount).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, uint64(1000000000-vestingAmount)).
		Vesting(from.Addr, uint64(vestingAmount))

	bb = bb.
		Tx().Type(core.TxOpWithdrawVesting).Value(withdrawAmount).SignPair(from).Execute()

	bb.Expect().Vesting(from.Addr, vestingAmount-withdrawAmount)

	block := bb.Build()
	reservedTasks := block.State().GetReservedTasks()
	assert.Equal(t, core.RtWithdrawNum, len(reservedTasks))
	for i := 0; i < len(reservedTasks); i++ {
		t.Logf("No.%v ts:%v, payload:%v", i, reservedTasks[i].Timestamp(), reservedTasks[i].Payload())
		assert.Equal(t, core.RtWithdrawType, reservedTasks[i].TaskType())
		assert.Equal(t, from.Addr, reservedTasks[i].From())
		assert.Equal(t, bb.B.Timestamp()+int64(i+1)*core.RtWithdrawInterval, reservedTasks[i].Timestamp())
	}

	acc, err := bb.B.State().GetAccount(from.Addr)
	require.NoError(t, err)
	t.Logf("ts:%v, balance: %v", bb.B.Timestamp(), acc.Balance())

	for i := 0; i < len(reservedTasks)+5; i++ {
		bb = bb.SignMiner().
			ChildWithTimestamp(bb.B.Timestamp() + core.RtWithdrawInterval)

		acc, err := bb.B.State().GetAccount(from.Addr)
		require.NoError(t, err)
		reservedTasks = bb.B.State().GetReservedTasks()
		t.Logf("ts:%v, balance: %v, remain tasks: %v", bb.B.Timestamp(), acc.Balance(), len(reservedTasks))
	}

	bb.Expect().Balance(from.Addr, 1000000000-vestingAmount+withdrawAmount)
	assert.Equal(t, 0, len(reservedTasks))
}

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
	assert.Equal(t, cert.RevocationTime, int64(-1))
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
