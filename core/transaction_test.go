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
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
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
		Tx().StakeTx(from, 10000000000000000).Execute().
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1000001000000000000).SignPair(from).ExecuteErr(core.ErrBalanceNotEnough).
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(10).SignPair(from).Execute().
		Expect().
		Balance(to.Addr, 10).
		Balance(from.Addr, 1000000000000000000-10-10000000000000000).
		Vesting(from.Addr, 10000000000000000)

}

func TestAddRecord(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	recordHash := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	payload := &core.AddRecordPayload{RecordHash: recordHash}
	owner := bb.TokenDist[0]

	block := bb.
		Tx().StakeTx(owner, 10000000000000000).Execute().
		Tx().Type(core.TxOpAddRecord).Payload(payload).SignPair(owner).Execute().
		Build()

	acc, err := block.State().GetAccount(owner.Addr)
	assert.NoError(t, err)

	recordBytes, err := acc.GetData(core.RecordsPrefix, recordHash)
	assert.NoError(t, err)

	pbRecord := new(corepb.Record)
	assert.NoError(t, proto.Unmarshal(recordBytes, pbRecord))
	assert.Equal(t, recordHash, pbRecord.RecordHash)
}

func TestVestAndWithdraw(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[testutil.DynastySize]
	vestingAmount := uint64(1000000000000000)
	withdrawAmount := uint64(301)

	bb = bb.
		Tx().Type(core.TxOpVest).Value(vestingAmount).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, uint64(1000000000000000000-vestingAmount)).
		Vesting(from.Addr, uint64(vestingAmount))

	bb = bb.
		Tx().Type(core.TxOpWithdrawVesting).Value(withdrawAmount).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, uint64(1000000000000000000-vestingAmount)).
		Vesting(from.Addr, vestingAmount-withdrawAmount).
		Unstaking(from.Addr, withdrawAmount)

	acc, err := bb.B.State().GetAccount(from.Addr)
	require.NoError(t, err)
	t.Logf("ts:%v, balance: %v", bb.B.Timestamp(), acc.Balance)

	bb = bb.SignMiner().ChildWithTimestamp(bb.B.Timestamp() + int64(core.UnstakingWaitDuration/time.Second) + 1).
		Tx().Type(core.TxOpAddRecord).Payload(&core.AddRecordPayload{}).SignPair(from).Execute()
	bb.Expect().
		Balance(from.Addr, 1000000000000000000-vestingAmount+withdrawAmount).
		Unstaking(from.Addr, 0)
}

func TestAddAndRevokeCertification(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	issuer := bb.TokenDist[0]
	certified := bb.TokenDist[1]

	issueTime := time.Now().Unix()
	expirationTime := time.Now().Unix() + int64(100000)
	hash := byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")

	// Add certification Test
	addPayload := &core.AddCertificationPayload{
		IssueTime:       issueTime,
		ExpirationTime:  expirationTime,
		CertificateHash: hash,
	}

	bb = bb.Stake().
		Tx().Type(core.TxOpAddCertification).To(certified.Addr).Payload(addPayload).CalcHash().SignPair(issuer).Execute()

	block := bb.PayReward().Seal().Build()

	issuerAcc, err := block.State().GetAccount(issuer.Addr)
	require.NoError(t, err)
	certifiedAcc, err := block.State().GetAccount(certified.Addr)
	require.NoError(t, err)

	certBytes0, err := issuerAcc.GetData(core.CertIssuedPrefix, hash)
	assert.NoError(t, err)
	certBytes1, err := certifiedAcc.GetData(core.CertReceivedPrefix, hash)
	assert.NoError(t, err)
	require.True(t, byteutils.Equal(certBytes0, certBytes1))

	certBytes := certBytes0
	pbCert := new(corepb.Certification)
	require.NoError(t, proto.Unmarshal(certBytes, pbCert))

	assert.Equal(t, pbCert.CertificateHash, hash)
	assert.Equal(t, pbCert.Issuer, issuer.Addr.Bytes())
	assert.Equal(t, pbCert.Certified, certified.Addr.Bytes())
	assert.Equal(t, pbCert.IssueTime, issueTime)
	assert.Equal(t, pbCert.ExpirationTime, expirationTime)
	assert.Equal(t, pbCert.RevocationTime, int64(-1))
	t.Logf("Add certification test complete")

	// Revoke certification Test
	revokeTime := time.Now().Unix() + int64(50000)
	revokePayload := &core.RevokeCertificationPayload{CertificateHash: hash}

	bb = bb.
		Tx().Type(core.TxOpRevokeCertification).Payload(revokePayload).Timestamp(expirationTime + int64(1)).SignPair(issuer).ExecuteErr(core.ErrCertAlreadyExpired).
		Tx().Type(core.TxOpRevokeCertification).Payload(revokePayload).Timestamp(revokeTime).SignPair(issuer).Execute().
		Tx().Type(core.TxOpRevokeCertification).Payload(revokePayload).Timestamp(revokeTime).SignPair(issuer).
		ExecuteErr(core.ErrCertAlreadyRevoked)
	block = bb.Build()

	issuerAcc, err = block.State().GetAccount(issuer.Addr)
	require.NoError(t, err)
	certifiedAcc, err = block.State().GetAccount(certified.Addr)
	require.NoError(t, err)

	certBytes0, err = issuerAcc.GetData(core.CertIssuedPrefix, hash)
	assert.NoError(t, err)
	certBytes1, err = certifiedAcc.GetData(core.CertReceivedPrefix, hash)
	assert.NoError(t, err)
	require.True(t, byteutils.Equal(certBytes0, certBytes1))

	certBytes = certBytes0
	pbCert = new(corepb.Certification)
	require.NoError(t, proto.Unmarshal(certBytes, pbCert))

	assert.Equal(t, pbCert.RevocationTime, revokeTime)
	t.Logf("Revoke certification test complete")

}
