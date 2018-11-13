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

	from := bb.TokenDist[testutil.DynastySize]
	to := testutil.NewAddrKeyPair(t)

	bb.
		Tx().StakeTx(from, 10000).Execute().
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(400000001).SignPair(from).ExecuteErr(core.ErrBalanceNotEnough).
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(10).SignPair(from).Execute().
		Expect().
		Balance(to.Addr, 10).
		Balance(from.Addr, 399989990).
		Vesting(from.Addr, 10000)

}

func TestAddRecord(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	recordHash := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	payload := &core.AddRecordPayload{RecordHash: recordHash}
	owner := bb.TokenDist[0]

	block := bb.
		Tx().StakeTx(owner, 10000).Execute().
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
	vestingAmount := 1000.0
	withdrawAmount := 301.0

	bb = bb.
		Tx().Type(core.TxOpVest).Value(vestingAmount).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, 400000000-vestingAmount).
		Vesting(from.Addr, vestingAmount)

	bb = bb.
		Tx().Type(core.TxOpWithdrawVesting).Value(withdrawAmount).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, 400000000-vestingAmount).
		Vesting(from.Addr, vestingAmount-withdrawAmount).
		Unstaking(from.Addr, withdrawAmount)

	acc, err := bb.B.State().GetAccount(from.Addr)
	require.NoError(t, err)
	t.Logf("ts:%v, balance: %v", bb.B.Timestamp(), acc.Balance)

	bb = bb.SignProposer().ChildWithTimestamp(bb.B.Timestamp() + int64(core.UnstakingWaitDuration/time.Second) + 1).
		Tx().Type(core.TxOpAddRecord).Payload(&core.AddRecordPayload{}).SignPair(from).Execute()
	bb.Expect().
		Balance(from.Addr, 400000000-vestingAmount+withdrawAmount).
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

	block := bb.PayReward().Flush().Seal().Build()

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
		Tx().Type(core.TxOpRevokeCertification).Payload(revokePayload).Timestamp(expirationTime + int64(1)).SignPair(
		issuer).ExecuteErr(core.ErrCertAlreadyExpired).
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

func TestPayerSigner(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	payer := bb.TokenDist[testutil.DynastySize]
	from := testutil.NewAddrKeyPair(t)
	to := testutil.NewAddrKeyPair(t)
	bb = bb.
		Tx().StakeTx(payer, 10000).Execute().
		Tx().Type(core.TxOpTransfer).To(from.Addr).Value(1000).SignPair(payer).Execute().
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(300).SignPair(from).ExecuteErr(core.ErrBandwidthNotEnough).
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(300).SignPair(from).SignPayerKey(payer.PrivKey).Execute()
	bb.
		Expect().
		Balance(to.Addr, 300).
		Balance(from.Addr, 700).
		Vesting(from.Addr, 0)

	payerAcc, err := bb.B.State().GetAccount(payer.Addr)
	require.NoError(t, err)

	//require.NoError(t,payerAcc.UpdateBandwidth(bb.B.Timestamp()))

	t.Log("Payer's bandwidth after payer sign", payerAcc.Bandwidth)

}

func TestRegisterAndDeregisterAlias(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()
	from := bb.TokenDist[testutil.DynastySize]
	const (
		collateralAmount = 1000
		testAliasName    = "testalias"
	)

	bb = bb.
		Tx().StakeTx(from, 10000).Execute().
		Tx().Type(core.TxOpRegisterAlias).
		Value(collateralAmount).
		SignPair(from).
		Payload(&core.RegisterAliasPayload{AliasName: testAliasName}).
		Execute()

	bb = bb.
		Tx().StakeTx(from, 10000).Execute().
		Tx().Type(core.TxOpRegisterAlias).
		Value(collateralAmount).
		SignPair(from).
		Payload(&core.RegisterAliasPayload{AliasName: testAliasName}).
		ExecuteErr(core.ErrAlreadyHaveAlias)

	bb.Expect().
		Balance(from.Addr, 400000000-collateralAmount-20000)
	acc, err := bb.B.State().AccState().GetAliasAccount(testAliasName)
	if err != nil {
		t.Log(err)
	}
	t.Logf("ts:%v, Account: %v", bb.B.Timestamp(), acc.Account)

	acc2, err := bb.B.State().AccState().GetAccount(from.Addr)
	aliasBytes, err := acc2.GetData(core.AliasPrefix, []byte("alias"))
	pbAlias := new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		t.Log(err)
	}
	t.Log(pbAlias.AliasName)

	bb = bb.
		Tx().Type(core.TxOpDeregisterAlias).
		SignPair(from).
		Execute()

	bb = bb.
		Tx().Type(core.TxOpDeregisterAlias).
		SignPair(from).
		ExecuteErr(core.ErrAliasNotExist)

	bb.Expect().
		Balance(from.Addr, 400000000-20000)

	acc, err = bb.B.State().AccState().GetAliasAccount(testAliasName)
	//require.NoError(t, err)
	if err != nil {
		t.Log(err)
	} else {
		t.Logf("ts:%v, Account: %v", bb.B.Timestamp(), acc.Account)
	}
	acc2, err = bb.B.State().AccState().GetAccount(from.Addr)
	aliasBytes, err = acc2.GetData(core.AliasPrefix, []byte(core.AliasKey))
	pbAlias = new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		t.Log(err)
	}
	t.Log(pbAlias.AliasName)
}
