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

	bb := blockutil.New(t,testutil.DynastySize).Block(seed.Tail()).
		ExecuteErr(transaction1,core.ErrBalanceNotEnough).
		ExecuteTx(transaction2).
		ExecuteErr(transaction3,dpos.ErrAlreadyCandidate)
	bb.Expect().Balance(candidate.Addr,uint64(1000000000-10))
	block := bb.Build()

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

	bb = bb.ExecuteTx(transactionQuit1).
		ExecuteErr(transactionQuit2, dpos.ErrNotCandidate)
	bb.Expect().Balance(candidate.Addr,uint64(1000000000))
	block = bb.Build()

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

	transactionBecome := blockutil.New(t,testutil.DynastySize).Tx().ChainID(testutil.ChainID).From(candidate.Addr).
		Value(10).Nonce(1).Type(dpos.TxOperationBecomeCandidate).CalcHash().SignKey(candidate.PrivKey).Build()

	transactionVote := blockutil.New(t,testutil.DynastySize).Tx().ChainID(testutil.ChainID).From(voter.Addr).
		To(candidate.Addr).Nonce(2).Type(dpos.TxOperationVote).CalcHash().SignKey(voter.PrivKey).Build()

	bb := blockutil.New(t,testutil.DynastySize).Block(seed.Tail()).
		ExecuteTx(transactionVest).
		ExecuteTx(transactionBecome).
		ExecuteTx(transactionVote)
	bb.Expect().Balance(candidate.Addr,uint64(1000000000-10))
	block := bb.Build()

	voterAcc, err := block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	assert.Equal(t, voterAcc.Voted(), candidate.Addr.Bytes())

	candidateBytes, err := block.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.NoError(t, err)

	candidatePb := new(dpospb.Candidate)
	proto.Unmarshal(candidateBytes, candidatePb)

	expectedVotePower, err := util.NewUint128FromUint(333).ToFixedSizeByteSlice()
	require.NoError(t, err)
	assert.Equal(t, candidatePb.VotePower, expectedVotePower)

}