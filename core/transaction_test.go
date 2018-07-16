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
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	from := bb.TokenDist[0]

	bb = bb.
		Tx().Type(core.TxOpVest).Value(333).Nonce(1).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, uint64(1000000000-333)).
		Vesting(from.Addr, uint64(333))

	bb = bb.
		Tx().Type(core.TxOpWithdrawVesting).Value(133).Nonce(1).SignPair(from).Execute()

	bb.Expect().Vesting(from.Addr, uint64(333-133))

	block := bb.Build()
	reservedTask := block.State().GetReservedTasks()
	assert.Equal(t, 3, len(reservedTask))
	for i := 0; i < len(reservedTask); i++ {
		assert.Equal(t, core.RtWithdrawType, reservedTask[i].TaskType())
		assert.Equal(t, from.Addr, reservedTask[i].From())
		assert.Equal(t, bb.B.Timestamp()+int64(i+1)*core.RtWithdrawInterval, reservedTask[i].Timestamp())
	}
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
