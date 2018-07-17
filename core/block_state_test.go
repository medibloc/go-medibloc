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
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneState(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()
	block := bb.Build()

	state := block.State()
	newState, err := state.Clone()
	require.NoError(t, err)

	assert.Equal(t, state.AccountsRoot(), newState.AccountsRoot())
	assert.Equal(t, state.TransactionsRoot(), newState.TransactionsRoot())
	assert.Equal(t, state.UsageRoot(), newState.UsageRoot())
	assert.Equal(t, state.RecordsRoot(), newState.RecordsRoot())

	ds, err := state.DposState().RootBytes()
	newDs, err := newState.DposState().RootBytes()

	assert.Equal(t, ds, newDs)
	assert.Equal(t, state.CertificationRoot(), newState.CertificationRoot())
	assert.Equal(t, state.ReservationQueueHash(), newState.ReservationQueueHash())

}

func TestDynastyState(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	actual, err := dpos.DynastyStateToDynasty(bb.Build().State().DposState().DynastyState())
	assert.NoError(t, err)

	for i, v := range actual {
		assert.Equal(t, bb.Dynasties[i].Addr, *v)
	}
}

func TestNonceCheck(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[0]
	to := testutil.NewAddrKeyPair(t)

	bb = bb.
		Tx().Type(core.TxOpSend).Value(10).To(to.Addr).Nonce(0).From(from.Addr).CalcHash().SignKey(from.PrivKey).ExecuteErr(core.ErrSmallTransactionNonce).
		Tx().Type(core.TxOpSend).Value(10).To(to.Addr).Nonce(1).SignPair(from).Execute().
		Tx().Type(core.TxOpSend).Value(10).To(to.Addr).Nonce(3).SignPair(from).ExecuteErr(core.ErrLargeTransactionNonce).
		Tx().Type(core.TxOpSend).Value(10).To(to.Addr).SignPair(from).Execute().
		Tx().Type(core.TxOpSend).Value(10).To(to.Addr).SignPair(from).Execute()
}

func TestUpdateUsage(t *testing.T) {
	nextMintTs := dpos.NextMintSlot2(time.Now().Unix())
	bb := blockutil.New(t, testutil.DynastySize).Genesis().ChildWithTimestamp(nextMintTs)

	from := bb.TokenDist[0]
	to := bb.TokenDist[1]

	txs := core.Transactions{
		bb.Tx().Type(core.TxOpSend).To(to.Addr).Value(10).Nonce(1).SignPair(from).Build(),
		bb.Tx().Type(core.TxOpSend).To(to.Addr).Value(20).Nonce(2).SignPair(from).Build(),
		bb.Tx().Type(core.TxOpSend).To(to.Addr).Value(20).Nonce(3).Timestamp(0).SignPair(from).Build(),
	}

	bb = bb.
		ExecuteTx(txs[0]).
		ExecuteTx(txs[1]).
		ExecuteTxErr(txs[2], core.ErrTooOldTransaction)

	block := bb.Build()

	usage, err := block.State().GetUsage(from.Addr)
	assert.NoError(t, err)

	for i, tx := range usage {
		assert.Equal(t, tx.Hash, txs[i].Hash())
		assert.Equal(t, tx.Timestamp, txs[i].Timestamp())
	}
}

func TestPayerUsageUpdate(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	user := testutil.NewAddrKeyPair(t)
	payer := bb.TokenDist[0]

	recordHash := byteutils.Hex2Bytes("03e7b794e1de1851b52ab0b0b995cc87558963265a7b26630f26ea8bb9131a7e")
	payload := core.NewAddRecordPayload(recordHash)
	bb = bb.Tx().Type(core.TxOpAddRecord).Payload(payload).SignPair(user).SignPayerKey(payer.PrivKey).Execute()
	block := bb.Build()

	usage, err := block.State().GetUsage(payer.Addr)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(usage))
}
