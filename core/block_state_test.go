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

func TestUpdateBandwidth(t *testing.T) {
	nextMintTs := dpos.NextMintSlot2(time.Now().Unix())
	bb := blockutil.New(t, testutil.DynastySize).Genesis().ChildWithTimestamp(nextMintTs)

	from := bb.TokenDist[len(bb.TokenDist)-1]
	to := bb.TokenDist[len(bb.TokenDist)-2]

	tx := bb.Tx().StakeTx(from, 200).Build()
	consumed := blockutil.Bandwidth(t, tx, bb.Build())
	bb = bb.ExecuteTx(tx)
	bb.Expect().Balance(from.Addr, 400000000-200).Vesting(from.Addr, 200)

	tx = bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1).SignPair(from).Build()
	consumed, err := consumed.Add(blockutil.Bandwidth(t, tx, bb.Build()))
	require.NoError(t, err)
	bb = bb.ExecuteTx(tx).SignProposer()
	t.Log(bb.B.BlockData.Transactions())
	bb.Expect().Bandwidth(from.Addr, consumed)

	afterWeek := dpos.NextMintSlot2(nextMintTs + 7*24*60*60)
	tx = bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1).SignPair(from).Build()
	consumed = blockutil.Bandwidth(t, tx, bb.Build())
	bb.ChildWithTimestamp(afterWeek).
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1).SignPair(from).Execute().
		Expect().Bandwidth(from.Addr, consumed)
}

func TestUpdatePayerBandwidth(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	user := testutil.NewAddrKeyPair(t)
	payer := bb.TokenDist[0]

	recordHash := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	payload := &core.AddRecordPayload{
		RecordHash: recordHash,
	}

	txs := []*core.Transaction{
		bb.Tx().StakeTx(payer, 10000).Build(),
		bb.Tx().Type(core.TxOpAddRecord).Payload(payload).SignPair(user).SignPayerKey(payer.PrivKey).Build(),
	}

	consumed := util.NewUint128()
	for _, tx := range txs {
		var err error
		bb = bb.ExecuteTx(tx)
		consumed, err = consumed.Add(blockutil.Bandwidth(t, tx, bb.Build()))
		require.NoError(t, err)
	}
	bb.Expect().Bandwidth(payer.Addr, consumed).Bandwidth(user.Addr, util.NewUint128())
}

func TestBandwidthWhenUnstaking(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()
	from := bb.TokenDist[testutil.DynastySize]

	bb.Tx().StakeTx(from, 100).SignPair(from).Execute().
		Tx().Type(core.TxOpWithdrawVesting).Value(100).SignPair(from).ExecuteErr(core.ErrVestingNotEnough).
		Tx().Type(core.TxOpWithdrawVesting).Value(40).SignPair(from).Execute().
		Expect().
		Unstaking(from.Addr, 40).
		Vesting(from.Addr, 60)
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
		Tx().Type(core.TxOpVest).Value(100).SignPair(from).Execute().
		Tx().Type(core.TxOpWithdrawVesting).Value(100).SignPair(from).Execute().
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

func TestBandwidthUsageAndRef(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	bb = bb.Child().Stake().SignProposer()
	b1 := bb.Build()

	to := testutil.NewAddrKeyPair(t)
	from := bb.TokenDist[testutil.DynastySize]

	maxCPU, err := b1.CPURef().Mul(util.NewUint128FromUint(uint64(core.CPULimit)))
	assert.NoError(t, err)

	bb = bb.Child()
	for i := 2; ; i++ {
		TX := bb.Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Build()
		tx, err := core.NewTransferTx(TX)
		assert.NoError(t, err)

		reqCPU, _, err := tx.Bandwidth(b1.State())
		assert.NoError(t, err)

		maxCPU, err = maxCPU.Sub(reqCPU)
		if err != nil {
			bb.ExecuteTxErr(TX, core.ErrExceedBlockMaxCPUUsage)
			break
		}
		bb.ExecuteTx(TX)
	}

	b2 := bb.SignProposer().Build()

	maxNet, err := b2.CPURef().Mul(util.NewUint128FromUint(uint64(core.NetLimit)))
	assert.NoError(t, err)

	bb = bb.Child()
	payload := &core.DefaultPayload{
		Message: "QWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNMqwertyuiopasdfghjklzxcvbnm",
	}

	for i := 3; ; i++ {
		TX := bb.Tx().Type(core.TxOpTransfer).Payload(payload).Value(10).To(to.Addr).SignPair(from).
			Build()
		tx, err := core.NewTransferTx(TX)
		assert.NoError(t, err)

		_, reqNet, err := tx.Bandwidth(b2.State())
		assert.NoError(t, err)

		maxNet, err = maxNet.Sub(reqNet)
		if err != nil {
			bb.ExecuteTxErr(TX, core.ErrExceedBlockMaxNetUsage)
			break
		}
		bb.ExecuteTx(TX)
	}
	b3 := bb.SignProposer().Build()

	bb = bb.Child().SignProposer()
	b4 := bb.Build()

	bb = bb.Child().SignProposer()
	b5 := bb.Build()

	assert.True(t, b1.CPUUsage().Cmp(b2.CPUUsage()) < 0)
	assert.True(t, b2.NetUsage().Cmp(b3.NetUsage()) < 0)
	assert.True(t, b2.CPURef().Cmp(b3.CPURef()) < 0)
	assert.True(t, b3.NetRef().Cmp(b4.NetRef()) < 0)
	assert.True(t, b4.CPUUsage().Uint64() == 0)
	assert.True(t, b4.NetUsage().Uint64() == 0)
	assert.True(t, b5.CPURef().Cmp(b4.CPURef()) < 0)
	assert.True(t, b5.NetRef().Cmp(b4.NetRef()) < 0)
}
