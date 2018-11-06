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

	"github.com/medibloc/go-medibloc/common"
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
	block := bb.SignMiner().Build()

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

	actual, err := bb.Build().State().DposState().Dynasty()
	assert.NoError(t, err)

	for i, v := range actual {
		assert.Equal(t, bb.Dynasties[i].Addr, v)
	}
}

func TestNonceCheck(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[0]
	to := testutil.NewAddrKeyPair(t)

	bb = bb.
		Tx().StakeTx(from, 100000).Execute().
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).Nonce(1).From(from.Addr).CalcHash().SignKey(from.PrivKey).ExecuteErr(core.ErrSmallTransactionNonce).
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Execute().
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).Nonce(5).SignPair(from).ExecuteErr(core.ErrLargeTransactionNonce).
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Execute().
		Tx().Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from).Execute()
}

func TestUpdateBandwidth(t *testing.T) {
	nextMintTs := dpos.NextMintSlot2(time.Now().Unix())
	bb := blockutil.New(t, testutil.DynastySize).Genesis().ChildWithTimestamp(nextMintTs)

	from := bb.TokenDist[0]
	to := bb.TokenDist[1]

	tx := bb.Tx().StakeTx(from, 200).Build()
	consumed := blockutil.Bandwidth(t, tx, bb.Build())
	bb = bb.ExecuteTx(tx)
	bb.Expect().Balance(from.Addr, 400000000-200).Vesting(from.Addr, 200)

	tx = bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(1).SignPair(from).Build()
	consumed, err := consumed.Add(blockutil.Bandwidth(t, tx, bb.Build()))
	require.NoError(t, err)
	bb = bb.ExecuteTx(tx).SignMiner()
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
	from := bb.TokenDist[0]

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
		Tx().Type(dpos.TxOpBecomeCandidate).Value(0).SignPair(from).Execute().
		Tx().Type(dpos.TxOpVote).
		Payload(&dpos.VotePayload{
			Candidates: []common.Address{from.Addr},
		}).SignPair(from).Execute().
		Tx().Type(dpos.TxOpQuitCandidacy).SignPair(from).Execute()

	block := bb.Build()

	_, err := block.State().GetAccount(from.Addr)
	require.NoError(t, err)

	_, err = block.State().GetAccount(to.Addr)
	require.NoError(t, err)

}

func TestRefBandwidth(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	bb = bb.Child().SignMiner()
	b := bb.Build()
	t.Log(b.Height(), b.Reward(), b.Supply(), b.CPURef(), b.NetRef())

	bb = bb.Child().SignMiner()
	b = bb.Build()
	t.Log(b.Height(), b.Reward(), b.Supply(), b.CPURef(), b.NetRef())
}
