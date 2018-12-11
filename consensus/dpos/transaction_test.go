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
	"testing"

	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBecomeAndQuitCandidate(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	candidate := bb.TokenDist[testutil.DynastySize]

	txType := dpos.TxOpBecomeCandidate
	bb = bb.
		Tx().StakeTx(candidate, 100000).Execute().
		Tx().Type(txType).Value(400000001).SignPair(candidate).ExecuteErr(core.ErrAliasNotExist).
		Tx().Type(core.TxOpRegisterAlias).Value(1000000).Payload(&core.RegisterAliasPayload{AliasName: "testname"}).SignPair(candidate).Execute().
		Tx().Type(txType).Value(400000001).SignPair(candidate).ExecuteErr(core.ErrBalanceNotEnough).
		Tx().Type(txType).Value(999999).SignPair(candidate).ExecuteErr(dpos.ErrNotEnoughCandidateCollateral).
		Tx().Type(txType).Value(1000000).SignPair(candidate).Execute().
		Tx().Type(txType).Value(1000000).SignPair(candidate).ExecuteErr(dpos.ErrAlreadyCandidate)

	bb.Expect().
		Balance(candidate.Addr, 400000000-100000-1000000-1000000).
		Staking(candidate.Addr, 100000)

	block := bb.Build()

	acc, err := block.State().GetAccount(candidate.Addr)
	require.NoError(t, err)

	cId := acc.CandidateID
	require.NotNil(t, cId)

	cs := block.State().DposState().CandidateState()
	c := new(dpos.Candidate)
	require.NoError(t, cs.GetData(cId, c))

	assert.Equal(t, candidate.Addr.Hex(), c.Addr.Hex())
	assert.Equal(t, blockutil.FloatToUint128(t, 1000000), c.Collateral)

	bb = bb.
		Tx().Type(dpos.TxOpQuitCandidacy).SignPair(candidate).Execute().
		Tx().Type(dpos.TxOpQuitCandidacy).SignPair(candidate).ExecuteErr(dpos.ErrNotCandidate)

	bb.Expect().Balance(candidate.Addr, 400000000-100000-1000000)

	block = bb.Build()
	cs = block.State().DposState().CandidateState()

	require.Equal(t, trie.ErrNotFound, cs.GetData(cId, c))
}

func TestVote(t *testing.T) {
	dynastySize := 21
	bb := blockutil.New(t, dynastySize).Genesis().Child()

	newCandidate := bb.TokenDist[dynastySize]
	voter := bb.TokenDist[dynastySize+1]

	votePayload := new(dpos.VotePayload)
	overSizePayload := new(dpos.VotePayload)
	duplicatePayload := new(dpos.VotePayload)
	candidates := append(bb.TokenDist[:dpos.MaxVote], newCandidate)

	bb = bb.
		Tx().Type(core.TxOpStake).Value(333).SignPair(voter).Execute().
		Tx().StakeTx(newCandidate, 10000).Execute().
		Tx().Type(core.TxOpRegisterAlias).Value(1000000).Payload(&core.RegisterAliasPayload{AliasName: "testname"}).SignPair(newCandidate).Execute().
		Tx().Type(dpos.TxOpBecomeCandidate).Value(1000000).SignPair(newCandidate).Execute()

	bb.Expect().Balance(newCandidate.Addr, 400000000-10000-1000000-1000000)
	block := bb.Build()

	candidateIDs := make([][]byte, 0)
	for _, v := range candidates {
		acc, err := bb.Build().State().GetAccount(v.Addr)
		require.NoError(t, err)

		candidateIDs = append(candidateIDs, acc.CandidateID)
		_, err = block.State().DposState().CandidateState().Get(acc.CandidateID)
		require.NoError(t, err)
	}

	votePayload.CandidateIDs = candidateIDs[1:]
	overSizePayload.CandidateIDs = candidateIDs
	duplicatePayload.CandidateIDs = candidateIDs[:dpos.MaxVote]
	duplicatePayload.CandidateIDs[0] = candidateIDs[1]

	bb = bb.
		Tx().Type(dpos.TxOpVote).Payload(overSizePayload).SignPair(voter).ExecuteErr(dpos.ErrOverMaxVote).
		Tx().Type(dpos.TxOpVote).Payload(duplicatePayload).SignPair(voter).ExecuteErr(dpos.ErrDuplicateVote).
		Tx().Type(dpos.TxOpVote).Payload(votePayload).SignPair(voter).Execute()

	voterAcc, err := block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	for _, v := range votePayload.CandidateIDs {
		_, err = voterAcc.Voted.Get(v)
		assert.NoError(t, err)
	}

	for _, v := range candidateIDs[:dpos.MaxVote-1] {
		candidate := new(dpos.Candidate)
		require.NoError(t, block.State().DposState().CandidateState().GetData(v, candidate))
		assert.Equal(t, blockutil.FloatToUint128(t, 333+100000000), candidate.VotePower)
	}

	candidate := new(dpos.Candidate)
	require.NoError(t, block.State().DposState().CandidateState().GetData(candidateIDs[dpos.MaxVote], candidate))
	assert.Equal(t, blockutil.FloatToUint128(t, 333), candidate.VotePower)

	// Reset vote to nil
	bb = bb.
		Tx().Type(dpos.TxOpVote).Payload(&dpos.VotePayload{}).SignPair(voter).Execute()

	block = bb.Build()

	voterAcc, err = block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	for _, v := range votePayload.CandidateIDs {
		_, err = voterAcc.Voted.Get(v)
		assert.Error(t, trie.ErrNotFound)
	}

	for _, v := range candidateIDs[:dpos.MaxVote-1] {
		candidate := new(dpos.Candidate)
		require.NoError(t, block.State().DposState().CandidateState().GetData(v, candidate))
		assert.Equal(t, blockutil.FloatToUint128(t, 100000000), candidate.VotePower)
	}

	candidate = new(dpos.Candidate)
	require.NoError(t, block.State().DposState().CandidateState().GetData(candidateIDs[dpos.MaxVote], candidate))
	assert.Equal(t, blockutil.FloatToUint128(t, 0), candidate.VotePower)

}
