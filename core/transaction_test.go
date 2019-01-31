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
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"

	"github.com/gogo/protobuf/proto"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()
	from := bb.TokenDist[blockutil.DynastySize]
	to := keyutil.NewAddrKeyPair(t)

	bb.
		Tx().StakeTx(from, 10000).Execute().
		Tx().Type(transaction.TxOpTransfer).To(to.Addr).Value(400000001).SignPair(from).ExecuteErr(transaction.ErrBalanceNotEnough).
		Tx().Type(transaction.TxOpTransfer).To(to.Addr).Value(10).SignPair(from).Execute().
		Expect().
		Balance(to.Addr, 10).
		Balance(from.Addr, 199989990).
		Staking(from.Addr, 10000)

}

func TestAddRecord(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis()

	recordHash, err := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	require.NoError(t, err)
	payload := &transaction.AddRecordPayload{RecordHash: recordHash}
	owner := bb.TokenDist[0]

	block := bb.
		Tx().StakeTx(owner, 10000).Execute().
		Tx().Type(transaction.TxOpAddRecord).Payload(payload).SignPair(owner).Execute().
		Build()

	acc, err := block.State().GetAccount(owner.Addr)
	assert.NoError(t, err)

	recordBytes, err := acc.GetData(coreState.RecordsPrefix, recordHash)
	assert.NoError(t, err)

	pbRecord := new(corepb.Record)
	assert.NoError(t, proto.Unmarshal(recordBytes, pbRecord))
	assert.Equal(t, recordHash, pbRecord.RecordHash)
}

func TestStakingAndUnstaking(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[blockutil.DynastySize]
	stakingAmount := 1000.0
	unstakingAmount := 301.0

	bb = bb.
		Tx().Type(transaction.TxOpStake).Value(stakingAmount).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, 200000000-stakingAmount).
		Staking(from.Addr, stakingAmount)

	bb = bb.
		Tx().Type(transaction.TxOpUnstake).Value(unstakingAmount).SignPair(from).Execute()

	bb.Expect().
		Balance(from.Addr, 200000000-stakingAmount).
		Staking(from.Addr, stakingAmount-unstakingAmount).
		Unstaking(from.Addr, unstakingAmount)

	acc, err := bb.B.State().GetAccount(from.Addr)
	require.NoError(t, err)
	t.Logf("ts:%v, balance: %v", bb.B.Timestamp(), acc.Balance)

	recordHash, _ := byteutils.Hex2Bytes("255607ec7ef55d7cfd8dcb531c4aa33c4605f8aac0f5784a590041690695e6f7")
	payload := &transaction.AddRecordPayload{
		RecordHash: recordHash,
	}

	bb = bb.SignProposer().ChildWithTimestamp(bb.B.Timestamp() + int64(coreState.UnstakingWaitDuration/time.Second) + 1).
		Tx().Type(transaction.TxOpAddRecord).Payload(payload).SignPair(from).Execute()
	bb.Expect().
		Balance(from.Addr, 200000000-stakingAmount+unstakingAmount).
		Unstaking(from.Addr, 0)
}

func TestAddAndRevokeCertification(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()

	issuer := bb.TokenDist[0]
	certified := bb.TokenDist[1]

	issueTime := time.Now().Unix()
	expirationTime := time.Now().Unix() + int64(100000)
	hash, err := byteutils.Hex2Bytes("02e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	require.NoError(t, err)

	// Add certification Test
	addPayload := &transaction.AddCertificationPayload{
		IssueTime:       issueTime,
		ExpirationTime:  expirationTime,
		CertificateHash: hash,
	}

	bb = bb.Stake().
		Tx().Type(transaction.TxOpAddCertification).To(certified.Addr).Payload(addPayload).CalcHash().SignPair(issuer).Execute()

	proposer := bb.FindProposer()
	block := bb.Coinbase(proposer.Addr).PayReward().Flush().Seal().Build()

	issuerAcc, err := block.State().GetAccount(issuer.Addr)
	require.NoError(t, err)
	certifiedAcc, err := block.State().GetAccount(certified.Addr)
	require.NoError(t, err)

	certBytes0, err := issuerAcc.GetData(coreState.CertIssuedPrefix, hash)
	assert.NoError(t, err)
	certBytes1, err := certifiedAcc.GetData(coreState.CertReceivedPrefix, hash)
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
	revokePayload := &transaction.RevokeCertificationPayload{CertificateHash: hash}

	bb.ChildWithTimestamp(expirationTime + int64(1)).
		Tx().Type(transaction.TxOpRevokeCertification).Payload(revokePayload).SignPair(issuer).ExecuteErr(transaction.ErrCertAlreadyExpired)

	bb = bb.ChildWithTimestamp(revokeTime).
		Tx().Type(transaction.TxOpRevokeCertification).Payload(revokePayload).SignPair(issuer).Execute().
		Tx().Type(transaction.TxOpRevokeCertification).Payload(revokePayload).SignPair(issuer).
		ExecuteErr(transaction.ErrCertAlreadyRevoked)
	block = bb.Build()

	issuerAcc, err = block.State().GetAccount(issuer.Addr)
	require.NoError(t, err)
	certifiedAcc, err = block.State().GetAccount(certified.Addr)
	require.NoError(t, err)

	certBytes0, err = issuerAcc.GetData(coreState.CertIssuedPrefix, hash)
	assert.NoError(t, err)
	certBytes1, err = certifiedAcc.GetData(coreState.CertReceivedPrefix, hash)
	assert.NoError(t, err)
	require.True(t, byteutils.Equal(certBytes0, certBytes1))

	certBytes = certBytes0
	pbCert = new(corepb.Certification)
	require.NoError(t, proto.Unmarshal(certBytes, pbCert))

	assert.Equal(t, pbCert.RevocationTime, revokeTime)
	t.Logf("Revoke certification test complete")

}

func TestPayerSigner(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()

	payer := bb.TokenDist[blockutil.DynastySize]
	from := keyutil.NewAddrKeyPair(t)
	to := keyutil.NewAddrKeyPair(t)
	bb = bb.
		Tx().StakeTx(payer, 10000).Execute().
		Tx().Type(transaction.TxOpTransfer).To(from.Addr).Value(1000).SignPair(payer).Execute().
		Tx().Type(transaction.TxOpTransfer).To(to.Addr).Value(300).SignPair(from).ExecuteErr(core.ErrPointNotEnough).
		Tx().Type(transaction.TxOpTransfer).To(to.Addr).Value(300).SignPair(from).SignPayerPair(payer).Execute()
	bb.
		Expect().
		Balance(to.Addr, 300).
		Balance(from.Addr, 700).
		Staking(from.Addr, 0)

	payerAcc, err := bb.B.State().GetAccount(payer.Addr)
	require.NoError(t, err)

	// require.NoError(t,payerAcc.UpdatePoints(bb.B.Timestamp()))

	t.Log("Payer's points after payer sign", payerAcc.Points)

}

func TestTransaction_StakeWithPayerSign(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()

	payer := bb.TokenDist[blockutil.DynastySize]
	from := bb.TokenDist[blockutil.DynastySize+1]

	// Payer doesn't have point
	bb.Tx().StakeTx(from, 10000).SignPayerPair(payer).ExecuteErr(core.ErrPointNotEnough)

	// Payer has point
	bb.Tx().StakeTx(payer, 10000).Execute().
		Tx().StakeTx(from, 10000).SignPayerPair(payer).Execute()
}

func TestTransaction_UnstakeWithPayerSign(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()

	payer := bb.TokenDist[blockutil.DynastySize]
	from := bb.TokenDist[blockutil.DynastySize+1]

	// Payer doesn't have point
	bb.Tx().UnstakeTx(from, 10000).SignPayerPair(payer).ExecuteErr(core.ErrPointNotEnough)

	// Unable to unstake all point without payer sign
	bb.Tx().StakeTx(from, 10000).Execute().
		Tx().UnstakeTx(from, 10000).ExecuteErr(core.ErrPointNotEnough)

	// Partial unstake success
	bb.Tx().StakeTx(from, 10000).Execute().
		Tx().UnstakeTx(from, 9000).Execute()

	// Unstake with payer sign
	bb.Tx().StakeTx(from, 10000).Execute().
		Tx().StakeTx(payer, 10000).Execute().
		Tx().UnstakeTx(from, 10000).SignPayerPair(payer).Execute()
}

func TestRegisterAndDeregisterAlias(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()
	from := bb.TokenDist[blockutil.DynastySize]
	const (
		collateralAmount = 1000000
	)

	bb = bb.
		Tx().StakeTx(from, 10000).Execute().
		Tx().Type(transaction.TxOpRegisterAlias).
		Value(collateralAmount).
		SignPair(from).
		Payload(&transaction.RegisterAliasPayload{AliasName: testutil.TestAliasName}).
		Execute()

	bb = bb.
		Tx().StakeTx(from, 10000).Execute().
		Tx().Type(transaction.TxOpRegisterAlias).
		Value(collateralAmount).
		SignPair(from).
		Payload(&transaction.RegisterAliasPayload{AliasName: testutil.TestAliasName}).
		ExecuteErr(transaction.ErrAliasAlreadyHave)
	// Execute()

	bb.Expect().
		Balance(from.Addr, 200000000-collateralAmount-20000)
	acc, err := bb.B.State().GetAccountByAlias(testutil.TestAliasName)
	if err != nil {
		t.Log(err)
	}

	acc2, err := bb.B.State().GetAccount(from.Addr)
	aliasBytes, err := acc2.GetData("", []byte("alias"))
	pbAlias := new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		t.Log(err)
	}
	t.Log(pbAlias.AliasName)

	bb = bb.
		Tx().Type(transaction.TxOpDeregisterAlias).
		SignPair(from).
		Execute()

	bb = bb.
		Tx().Type(transaction.TxOpDeregisterAlias).
		SignPair(from).
		ExecuteErr(transaction.ErrAliasNotExist)

	bb.Expect().
		Balance(from.Addr, 200000000-20000)

	acc, err = bb.B.State().GetAccountByAlias(testutil.TestAliasName)
	// require.NoError(t, err)
	if err != nil {
		t.Log(err)
	} else {
		t.Logf("ts:%v, Account: %v", bb.B.Timestamp(), acc)
	}
	acc2, err = bb.B.State().GetAccount(from.Addr)
	aliasBytes, err = acc2.GetData("", []byte(coreState.AliasKey))
	pbAlias = new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		t.Log(err)
	}
	t.Log(pbAlias.AliasName)
}

func TestRegisterAliasTable(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()
	from := bb.TokenDist[blockutil.DynastySize]
	const (
		collateralAmount = 1000000
	)
	type aliasErrSet struct {
		name string
		err  error
	}
	testNames := []*aliasErrSet{
		{"", transaction.ErrAliasLengthUnderMinimum},
		{"accountAlias", transaction.ErrAliasInvalidChar},
		{"Testalias !@", transaction.ErrAliasInvalidChar},
		{"3accountalias", transaction.ErrAliasFirstLetter},
		{" accountalias", transaction.ErrAliasInvalidChar},
		{"accountalias01234", transaction.ErrAliasLengthExceedMaximum},
		{"accountalias!", transaction.ErrAliasInvalidChar},
		{"account_alias", transaction.ErrAliasInvalidChar},
		{"test	     as", transaction.ErrAliasInvalidChar},
		{"test메디블록", transaction.ErrAliasInvalidChar},
	}
	bb = bb.
		Tx().StakeTx(from, 10000).Execute()
	for _, es := range testNames {
		bb.Tx().Type(transaction.TxOpRegisterAlias).
			Value(collateralAmount).
			SignPair(from).
			Payload(&transaction.RegisterAliasPayload{AliasName: es.name}).
			ExecuteErr(es.err)
	}
}
