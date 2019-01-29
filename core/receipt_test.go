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
	dposState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	"github.com/medibloc/go-medibloc/core"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CASE 1. Receipt store and get from storage
// CASE 2. Get transaction from another node and check receipt
// CASE 3. Make invalid transaction and receipt should hold matched error message

func TestReceipt(t *testing.T) {
	network := testutil.NewNetwork(t)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	receiver := network.NewNode()
	receiver.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist).Block(seed.GenesisBlock()).ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix()))
	payer := seed.Config.TokenDist[network.DynastySize]

	tb := bb.Tx()
	tx1 := tb.Nonce(1).StakeTx(payer, 100).Build()
	tx2 := tb.Nonce(2).Type(coreState.TxOpUnstake).Value(200).SignPair(payer).
		Build()
	tx3 := tb.Nonce(2).Type(coreState.TxOpUnstake).Value(50).SignPair(payer).Build()
	b := bb.ExecuteTx(tx1).ExecuteTxErr(tx2, ErrStakingNotEnough).ExecuteTx(tx3).SignProposer().Build()

	require.NoError(t, seed.Med.BlockManager().PushBlockData(b.BlockData))
	require.NoError(t, seed.Med.BlockManager().BroadCast(b.BlockData))

	time.Sleep(1000 * time.Millisecond)
	tx1r, err := receiver.Tail().State().GetTx(tx1.Hash())
	assert.NoError(t, err)
	assert.True(t, tx1r.Receipt().Executed())
	assert.Equal(t, tx1r.Receipt().Error(), []byte(nil))

	tx2r, err := receiver.Tail().State().GetTx(tx2.Hash())
	assert.Equal(t, err, core.ErrNotFound)
	assert.Nil(t, tx2r)

	tx3r, err := receiver.Tail().State().GetTx(tx3.Hash())
	assert.NoError(t, err)
	assert.True(t, tx3r.Receipt().Executed())
	assert.Equal(t, tx3r.Receipt().Error(), []byte(nil))
}

func TestErrorTransactionReceipt(t *testing.T) {
	network := testutil.NewNetwork(t)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	receiver := network.NewNode()
	receiver.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, network.DynastySize).AddKeyPairs(seed.Config.TokenDist).
		Block(seed.GenesisBlock()).ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix()))
	payer := seed.Config.TokenDist[network.DynastySize]

	tb := bb.Tx()
	tx1 := tb.Nonce(1).StakeTx(payer, 1000).Build()

	recordHash, err := byteutils.Hex2Bytes("9eca7128409f609b2a72fc24985645665bbb99152b4b14261c3c3c93fb17cf54")
	require.NoError(t, err)
	payload := &transaction.AddRecordPayload{
		RecordHash: recordHash,
	}

	tx2 := tb.Nonce(2).Type(coreState.TxOpAddRecord).Payload(payload).SignPair(payer).Build()
	tx3 := tb.Nonce(3).Type(coreState.TxOpAddRecord).Payload(payload).SignPair(payer).Build()
	b := bb.ExecuteTx(tx1).ExecuteTx(tx2).ExecuteTxErr(tx3, transaction.ErrRecordAlreadyAdded).SignProposer().Build()

	require.NoError(t, seed.Med.BlockManager().PushBlockData(b.BlockData))
	require.NoError(t, seed.Med.BlockManager().BroadCast(b.BlockData))

	time.Sleep(1000 * time.Millisecond)
	tx1r, err := receiver.Tail().State().GetTx(tx1.Hash())
	assert.NoError(t, err)
	assert.True(t, tx1r.Receipt().Executed())
	assert.Equal(t, tx1r.Receipt().Error(), []byte(nil))

	tx2r, err := receiver.Tail().State().GetTx(tx2.Hash())
	assert.NoError(t, err)
	assert.True(t, tx2r.Receipt().Executed())
	assert.Equal(t, tx2r.Receipt().Error(), []byte(nil))

	tx3r, err := receiver.Tail().State().GetTx(tx3.Hash())
	assert.NoError(t, err)
	assert.False(t, tx3r.Receipt().Executed())
	assert.Equal(t, string(tx3r.Receipt().Error()[:]), transaction.ErrRecordAlreadyAdded.Error())
}

func TestWrongReceipt(t *testing.T) {
	network := testutil.NewNetwork(t)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, network.DynastySize).AddKeyPairs(seed.Config.TokenDist).
		Block(seed.GenesisBlock()).Child()

	b := bb.Tx().RandomTx().Execute().SignProposer().Build()

	b.Transactions()[0].Receipt().SetPoints(util.NewUint128())

	require.NoError(t, seed.Med.BlockManager().PushBlockData(b.BlockData))
	require.Error(t, testutil.ErrExecutionTimeout, seed.WaitUntilTailHeight(b.Height(), 1*time.Second))
}

func TestVoteTransactionReceipt(t *testing.T) {
	network := testutil.NewNetwork(t)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	receiver := network.NewNode()
	receiver.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, network.DynastySize).AddKeyPairs(seed.Config.TokenDist).
		Block(seed.GenesisBlock()).ChildWithTimestamp(dpos.
		NextMintSlot2(time.Now().Unix())).Stake()

	payer := seed.Config.TokenDist[network.DynastySize]
	payer2 := seed.Config.TokenDist[network.DynastySize+1]
	aliasPayload1 := &transaction.RegisterAliasPayload{AliasName: "helloworld10"}
	aliasPayload2 := &transaction.RegisterAliasPayload{AliasName: "helloworld20"}
	aliasTx1 := bb.Tx().Nonce(2).Payload(aliasPayload1).Value(1000000).Type(coreState.TxOpRegisterAlias).SignPair(payer).Build()
	aliasTx2 := bb.Tx().Nonce(2).Payload(aliasPayload2).Value(1000000).Type(coreState.TxOpRegisterAlias).SignPair(payer2).Build()

	candidateTx1 := bb.Tx().Value(1000000).Nonce(3).Type(dposState.TxOpBecomeCandidate).SignPair(payer).Build()
	candidateTx2 := bb.Tx().Value(1000000).Nonce(3).Type(dposState.TxOpBecomeCandidate).SignPair(payer2).Build()

	candidateId, _ := byteutils.Hex2Bytes(
		"e81217e7d3c1977b26f0d351f3ba2b8bbd3ab655a23e5142779a224e46e55417")
	invalidPayload := &transaction.VotePayload{
		CandidateIDs: [][]byte{candidateId, candidateTx1.Hash()},
	}
	invalidTx := bb.Tx().Nonce(4).Type(dposState.TxOpVote).Payload(invalidPayload).SignPair(payer).Build()

	validPayload := &transaction.VotePayload{
		CandidateIDs: [][]byte{candidateTx1.Hash(), candidateTx2.Hash()},
	}
	validTx := bb.Tx().Nonce(5).Type(dposState.TxOpVote).Payload(validPayload).SignPair(payer).Build()

	b := bb.
		ExecuteTx(aliasTx1).
		ExecuteTx(aliasTx2).
		ExecuteTx(candidateTx1).
		ExecuteTx(candidateTx2).
		ExecuteTxErr(invalidTx, transaction.ErrNotCandidate).
		ExecuteTx(validTx).
		SignProposer().Build()

	require.NoError(t, seed.Med.BlockManager().PushBlockData(b.BlockData))
	require.NoError(t, seed.Med.BlockManager().BroadCast(b.BlockData))

	time.Sleep(1000 * time.Millisecond)
	invalidTxr, err := receiver.Tail().State().GetTx(invalidTx.Hash())
	assert.NoError(t, err)
	assert.False(t, invalidTxr.Receipt().Executed())
	assert.Equal(t, transaction.ErrNotCandidate.Error(), string(invalidTxr.Receipt().Error()))

	time.Sleep(1000 * time.Millisecond)
	validTxr, err := receiver.Tail().State().GetTx(validTx.Hash())
	assert.NoError(t, err)
	assert.True(t, validTxr.Receipt().Executed())
	assert.Equal(t, "", string(validTxr.Receipt().Error()))
}
