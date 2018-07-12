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
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package dpos_test

import (
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/medibloc/go-medibloc/util/testutil"
	"testing"
	"github.com/medibloc/go-medibloc/util"
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common/trie"
)

func TestBecomeAndQuitCandidate(t *testing.T) {
	nt := testutil.NewNetwork(t, testutil.DynastySize)
	defer nt.Cleanup()
	seed := nt.NewSeedNode()
	seed.Start()

	candidate := seed.Config.TokenDist[testutil.DynastySize]

	tb := blockutil.New(t,testutil.DynastySize).Tx().ChainID(testutil.ChainID).From(candidate.Addr).Nonce(1).
		Type(dpos.TxOperationBecomeCandidate)

	transaction1 := tb.Value(1000000001).CalcHash().SignKey(candidate.PrivKey).Build()

	transaction2 := tb.Value(10).CalcHash().SignKey(candidate.PrivKey).Build()

	transaction3 := tb.Value(10).Nonce(2).CalcHash().SignKey(candidate.PrivKey).Build()

	newTxFunc := testutil.TxMap[transaction1.Type()]
	tx1, err := newTxFunc(transaction1)
	require.NoError(t, err)
	tx2, err := newTxFunc(transaction2)
	require.NoError(t, err)
	tx3, err := newTxFunc(transaction3)
	require.NoError(t, err)

	block := seed.Tail()

	block.State().BeginBatch()
	assert.Equal(t, core.ErrBalanceNotEnough, tx1.Execute(block))
	assert.NoError(t, tx2.Execute(block))
	assert.Equal(t, dpos.ErrAlreadyCandidate, tx3.Execute(block))
	assert.NoError(t, block.State().AcceptTransaction(transaction2, block.Timestamp()))
	block.State().Commit()

	acc, err := block.State().GetAccount(candidate.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-10)))
	candidateBytes, err := block.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.NoError(t, err)

	candidatePb := new(dpospb.Candidate)
	proto.Unmarshal(candidateBytes, candidatePb)

	tenBytes, err := util.NewUint128FromUint(10).ToFixedSizeByteSlice()
	assert.NoError(t, err)
	assert.Equal(t, candidatePb.Collatral, tenBytes)

	tbQuit := blockutil.New(t,testutil.DynastySize).Tx().ChainID(testutil.ChainID).From(candidate.Addr).Nonce(2).
		Type(dpos.TxOperationQuitCandidacy)

	transactionQuit1 := tbQuit.CalcHash().SignKey(candidate.PrivKey).Build()
	transactionQuit2 := tbQuit.Nonce(3).CalcHash().SignKey(candidate.PrivKey).Build()

	newTxFuncQuit := testutil.TxMap[transactionQuit1.Type()]
	txQuit1, err := newTxFuncQuit(transactionQuit1)
	txQuit2, err := newTxFuncQuit(transactionQuit2)
	require.NoError(t, err)

	block.State().BeginBatch()
	assert.NoError(t, txQuit1.Execute(block))
	assert.Equal(t, dpos.ErrNotCandidate,txQuit2.Execute(block))
	assert.NoError(t, block.State().AcceptTransaction(transaction2, block.Timestamp()))
	block.State().Commit()

	acc, err = block.State().GetAccount(candidate.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000)))
	_, err = block.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.Equal(t, trie.ErrNotFound, err)
}

func TestVote(t *testing.T) {
	nt := testutil.NewNetwork(t, testutil.DynastySize)
	defer nt.Cleanup()
	seed := nt.NewSeedNode()
	seed.Start()

	candidate := seed.Config.TokenDist[testutil.DynastySize]
	voter := seed.Config.TokenDist[testutil.DynastySize+1]

	transactionVest := blockutil.New(t,testutil.DynastySize).Tx().ChainID(testutil.ChainID).From(voter.Addr).
		Value(333).Nonce(1).Type(core.TxOperationVest).CalcHash().SignKey(voter.PrivKey).Build()

	newTxFunc := testutil.TxMap[transactionVest.Type()]
	txVest, err := newTxFunc(transactionVest)
	require.NoError(t, err)

	transactionBecome := blockutil.New(t,testutil.DynastySize).Tx().ChainID(testutil.ChainID).From(candidate.Addr).
		Value(10).Nonce(1).Type(dpos.TxOperationBecomeCandidate).CalcHash().SignKey(candidate.PrivKey).Build()

	newTxFunc = testutil.TxMap[transactionBecome.Type()]
	txBecome, err := newTxFunc(transactionBecome)
	require.NoError(t, err)

	transactionVote := blockutil.New(t,testutil.DynastySize).Tx().ChainID(testutil.ChainID).From(voter.Addr).
		To(candidate.Addr).Nonce(2).Type(dpos.TxOperationVote).CalcHash().SignKey(voter.PrivKey).Build()

	newTxFunc = testutil.TxMap[transactionVote.Type()]
	txVote, err := newTxFunc(transactionVote)
	require.NoError(t, err)

	block := seed.Tail()

	block.State().BeginBatch()
	assert.NoError(t, txBecome.Execute(block))
	assert.NoError(t, block.State().AcceptTransaction(transactionBecome, block.Timestamp()))

	assert.NoError(t, txVest.Execute(block))
	assert.NoError(t, block.State().AcceptTransaction(transactionVest, block.Timestamp()))

	assert.NoError(t, txVote.Execute(block))
	assert.NoError(t, block.State().AcceptTransaction(transactionVote, block.Timestamp()))

	block.State().Commit()

	acc, err := block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(333)))

	candidateBytes, err := block.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.NoError(t, err)

	candidatePb := new(dpospb.Candidate)
	proto.Unmarshal(candidateBytes, candidatePb)

	expectedVotePower, err := util.NewUint128FromUint(333).ToFixedSizeByteSlice()
	require.NoError(t, err)
	assert.Equal(t, candidatePb.VotePower, expectedVotePower)

}