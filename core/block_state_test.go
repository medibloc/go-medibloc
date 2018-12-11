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

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneState(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()
	block := bb.SignProposer().Build()

	state := block.State()
	newState, err := state.Clone()
	require.NoError(t, err)

	accRoot, err := state.AccountsRoot()
	require.NoError(t, err)
	newAccRoot, err := newState.AccountsRoot()
	require.NoError(t, err)
	assert.Equal(t, accRoot, newAccRoot)

	txsRoot, err := state.TxsRoot()
	require.NoError(t, err)
	newTxsRoot, err := newState.TxsRoot()
	require.NoError(t, err)
	assert.Equal(t, txsRoot, newTxsRoot)

	dposRoot, err := state.DposRoot()
	require.NoError(t, err)
	newDposRoot, err := newState.DposRoot()
	require.NoError(t, err)
	assert.Equal(t, dposRoot, newDposRoot)
}

func TestDynastyState(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	ds := bb.Build().State().DposState()

	for _, v := range bb.Dynasties {
		in, err := ds.InDynasty(v.Addr)
		require.NoError(t, err)
		assert.True(t, in)
	}
}

func TestNonceCheck(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[len(bb.TokenDist)-1]
	to := testutil.NewAddrKeyPair(t)

	bb = bb.
		Tx().StakeTx(from, 100000).Execute().
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).Nonce(1).SignPair(from).ExecuteErr(core.ErrSmallTransactionNonce).
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Execute().
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).Nonce(5).SignPair(from).ExecuteErr(core.ErrLargeTransactionNonce).
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Execute().
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Execute()
}

func TestUpdatePoints(t *testing.T) {
	nextMintTs := dpos.NextMintSlot2(time.Now().Unix())
	bb := blockutil.New(t, testutil.DynastySize).Genesis().ChildWithTimestamp(nextMintTs)

	from := bb.TokenDist[len(bb.TokenDist)-1]
	to := bb.TokenDist[len(bb.TokenDist)-2]

	tx := bb.Tx().StakeTx(from, 200).Build()
	staking := tx.Value().DeepCopy()

	consumed := blockutil.Points(t, tx, bb.Build())
	bb = bb.ExecuteTx(tx)
	bb.Expect().Balance(from.Addr, 400000000-200).Staking(from.Addr, 200)

	tx = bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1).SignPair(from).Build()
	consumed, err := consumed.Add(blockutil.Points(t, tx, bb.Build()))
	require.NoError(t, err)
	bb = bb.ExecuteTx(tx).SignProposer()
	t.Log(bb.B.BlockData.Transactions())
	remain, err := staking.Sub(consumed)
	require.NoError(t, err)
	bb.Expect().Points(from.Addr, remain)

	afterWeek := dpos.NextMintSlot2(nextMintTs + 7*24*60*60)
	tx = bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1).SignPair(from).Build()
	consumed = blockutil.Points(t, tx, bb.Build())
	remain, err = staking.Sub(consumed)
	require.NoError(t, err)

	bb.ChildWithTimestamp(afterWeek).
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1).SignPair(from).Execute().
		Expect().Points(from.Addr, remain)
}

func TestUpdatePayerPoints(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	user := testutil.NewAddrKeyPair(t)
	payer := bb.TokenDist[0]

	recordHash := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	payload := &core.AddRecordPayload{
		RecordHash: recordHash,
	}

	tx := bb.Tx().Type(core.TxOpAddRecord).Payload(payload).SignPair(user).SignPayerKey(payer.PrivKey).Build()
	bb = bb.ExecuteTx(tx)

	consumed := blockutil.Points(t, tx, bb.Build())
	staking, err := util.NewUint128FromString("100000000000000000000")
	require.NoError(t, err)
	remain, err := staking.Sub(consumed)
	require.NoError(t, err)
	bb.Expect().Points(payer.Addr, remain).Points(user.Addr, util.NewUint128())
}

func TestBandwidthWhenUnstaking(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()
	from := bb.TokenDist[testutil.DynastySize]

	bb.Tx().StakeTx(from, 100).SignPair(from).Execute().
		Tx().Type(core.TxOpUnstake).Value(100).SignPair(from).ExecuteErr(core.ErrStakingNotEnough).
		Tx().Type(core.TxOpUnstake).Value(40).SignPair(from).Execute().
		Expect().
		Unstaking(from.Addr, 40).
		Staking(from.Addr, 60)
}

func TestTxsFromTxsTo(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	to := testutil.NewAddrKeyPair(t)
	from := bb.TokenDist[testutil.DynastySize]

	bb = bb.Stake().
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(100).SignPair(from).Execute().
		Tx().Type(core.TxOpAddRecord).
		Payload(&core.AddRecordPayload{
			RecordHash: hash([]byte("Record Hash")),
		}).SignPair(from).Execute().
		Tx().Type(core.TxOpAddCertification).To(to.Addr).
		Payload(&core.AddCertificationPayload{
			IssueTime:       time.Now().Unix(),
			ExpirationTime:  time.Now().Add(24 * time.Hour * 365).Unix(),
			CertificateHash: hash([]byte("Certificate Root Hash")),
		}).SignPair(from).Execute().
		Tx().Type(core.TxOpRevokeCertification).To(to.Addr).
		Payload(&core.RevokeCertificationPayload{
			CertificateHash: hash([]byte("Certificate Root Hash")),
		}).SignPair(from).Execute().
		Tx().Type(core.TxOpStake).Value(100).SignPair(from).Execute().
		Tx().Type(core.TxOpUnstake).Value(100).SignPair(from).Execute().
		Tx().Type(core.TxOpRegisterAlias).Value(1000000).Payload(&core.RegisterAliasPayload{AliasName: "testname"}).SignPair(from).Execute().
		Tx().Type(dpos.TxOpBecomeCandidate).Value(1000000).SignPair(from).Execute().
		Tx().Type(dpos.TxOpVote).
		Payload(&dpos.VotePayload{
			CandidateIDs: [][]byte{},
		}).SignPair(from).Execute().
		Tx().Type(dpos.TxOpQuitCandidacy).SignPair(from).Execute()

	block := bb.Build()

	_, err := block.State().GetAccount(from.Addr)
	require.NoError(t, err)

	_, err = block.State().GetAccount(to.Addr)
	require.NoError(t, err)

}

func TestBandwidthUsageAndPrice(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	bb = bb.Child().Stake().SignProposer()

	to := testutil.NewAddrKeyPair(t)
	from := bb.TokenDist[testutil.DynastySize]

	bb = bb.Child()
	for i := 0; i < 3000; i++ {
		tx := bb.Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Build()
		bb = bb.ExecuteTx(tx)
	}
	tx := bb.Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Build()
	bb = bb.ExecuteTxErr(tx, core.ErrExceedBlockMaxCPUUsage)
	bb = bb.SignProposer()
	b1 := bb.Build()

	t.Log("Number of txs:", len(b1.Transactions()))
	t.Log("CPU usage:", b1.CPUUsage())
	t.Log("Net usage:", b1.NetUsage())
	assert.Equal(t, uint64(core.CPULimit), b1.CPUUsage())

	bb = bb.Child()

	dummyMsg := byteutils.Bytes2Hex(make([]byte, 2000))
	payload := &core.DefaultPayload{
		Message: dummyMsg,
	}

	remainNet := core.NetLimit
	for {
		tx := bb.Tx().Type(core.TxOpTransfer).Payload(payload).Value(10).To(to.Addr).SignPair(from).Build()
		net, err := tx.Size()
		require.NoError(t, err)
		bb = bb.ExecuteTx(tx)
		remainNet -= net
		if remainNet < net {
			break
		}
	}
	tx = bb.Tx().Type(core.TxOpTransfer).Payload(payload).Value(10).To(to.Addr).SignPair(from).Build()
	net, err := tx.Size()
	require.NoError(t, err)

	bb = bb.ExecuteTxErr(tx, core.ErrExceedBlockMaxNetUsage)
	bb = bb.SignProposer()
	b2 := bb.Build()

	t.Log("Single Tx net:", net)
	t.Log("Number of txs:", len(b2.Transactions()))
	t.Log("CPU usage:", b2.CPUUsage())
	t.Log("Net usage:", b2.NetUsage())
	assert.True(t, core.NetLimit < b2.NetUsage()+uint64(net))

	bb = bb.Child().SignProposer()
	b3 := bb.Build()

	assert.Equal(t, uint64(0), b3.CPUUsage())
	assert.Equal(t, uint64(0), b3.NetUsage())

	expect, err := b1.CPUPrice().MulWithRat(core.BandwidthIncreaseRate)
	assert.Equal(t, expect.String(), b2.CPUPrice().String())

	expect, err = b2.NetPrice().MulWithRat(core.BandwidthIncreaseRate)
	assert.Equal(t, expect.String(), b3.NetPrice().String())
}
