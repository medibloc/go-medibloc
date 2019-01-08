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
	dState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	cState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBecomeAndQuitCandidate(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	candidate := bb.TokenDist[testutil.DynastySize]

	txType := dState.TxOpBecomeCandidate
	bb = bb.
		Tx().StakeTx(candidate, 100000).Execute().
		Tx().Type(txType).Value(400000001).SignPair(candidate).ExecuteErr(transaction.ErrAliasNotExist).
		Tx().Type(cState.TxOpRegisterAlias).Value(1000000).Payload(&transaction.RegisterAliasPayload{AliasName: testutil.TestAliasName}).SignPair(candidate).Execute().
		Tx().Type(txType).Value(400000001).SignPair(candidate).ExecuteErr(transaction.ErrBalanceNotEnough).
		Tx().Type(txType).Value(999999).SignPair(candidate).ExecuteErr(transaction.ErrNotEnoughCandidateCollateral).
		Tx().Type(txType).Value(1000000).SignPair(candidate).Execute().
		Tx().Type(txType).Value(1000000).SignPair(candidate).ExecuteErr(transaction.ErrAlreadyCandidate)

	bb.Expect().
		Balance(candidate.Addr, 400000000-100000-1000000-1000000).
		Staking(candidate.Addr, 100000)

	block := bb.Build()

	acc, err := block.State().GetAccount(candidate.Addr)
	require.NoError(t, err)

	cId := acc.CandidateID
	require.NotNil(t, cId)

	c, err := block.State().DposState().GetCandidate(cId)
	require.NoError(t, err)

	assert.Equal(t, candidate.Addr.Hex(), c.Addr.Hex())
	assert.Equal(t, blockutil.FloatToUint128(t, 1000000), c.Collateral)

	bb = bb.
		Tx().Type(dState.TxOpQuitCandidacy).SignPair(candidate).Execute().
		Tx().Type(dState.TxOpQuitCandidacy).SignPair(candidate).ExecuteErr(transaction.ErrNotCandidate)

	bb.Expect().Balance(candidate.Addr, 400000000-100000-1000000)

	block = bb.Build()
	_, err = block.State().DposState().GetCandidate(cId)

	require.Equal(t, trie.ErrNotFound, err)
}

func TestVote(t *testing.T) {
	dynastySize := 21
	bb := blockutil.New(t, dynastySize).Genesis().Child()

	newCandidate := bb.TokenDist[dynastySize]
	voter := bb.TokenDist[dynastySize+1]

	votePayload := new(transaction.VotePayload)
	overSizePayload := new(transaction.VotePayload)
	duplicatePayload := new(transaction.VotePayload)
	candidates := append(bb.TokenDist[:transaction.VoteMaximum], newCandidate)

	bb = bb.
		Tx().Type(cState.TxOpStake).Value(333).SignPair(voter).Execute().
		Tx().StakeTx(newCandidate, 10000).Execute().
		Tx().Type(cState.TxOpRegisterAlias).Value(1000000).Payload(&transaction.RegisterAliasPayload{AliasName: testutil.TestAliasName}).SignPair(newCandidate).Execute().
		Tx().Type(dState.TxOpBecomeCandidate).Value(1000000).SignPair(newCandidate).Execute()

	bb.Expect().Balance(newCandidate.Addr, 400000000-10000-1000000-1000000)
	block := bb.Build()

	candidateIDs := make([][]byte, 0)
	for _, v := range candidates {
		acc, err := bb.Build().State().GetAccount(v.Addr)
		require.NoError(t, err)

		candidateIDs = append(candidateIDs, acc.CandidateID)
		_, err = block.State().DposState().GetCandidate(acc.CandidateID)
		require.NoError(t, err)
	}

	votePayload.CandidateIDs = candidateIDs[1:]
	overSizePayload.CandidateIDs = candidateIDs
	duplicatePayload.CandidateIDs = candidateIDs[:transaction.VoteMaximum]
	duplicatePayload.CandidateIDs[0] = candidateIDs[1]

	bb = bb.
		Tx().Type(dState.TxOpVote).Payload(overSizePayload).SignPair(voter).ExecuteErr(transaction.ErrOverMaxVote).
		Tx().Type(dState.TxOpVote).Payload(duplicatePayload).SignPair(voter).ExecuteErr(transaction.ErrDuplicateVote).
		Tx().Type(dState.TxOpVote).Payload(votePayload).SignPair(voter).Execute()

	voterAcc, err := block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	for _, v := range votePayload.CandidateIDs {
		_, err = voterAcc.Voted.Get(v)
		assert.NoError(t, err)
	}

	for _, v := range candidateIDs[:transaction.VoteMaximum-1] {
		candidate, err := block.State().DposState().GetCandidate(v)
		require.NoError(t, err)
		assert.Equal(t, blockutil.FloatToUint128(t, 333+100000000), candidate.VotePower)
	}

	candidate, err := block.State().DposState().GetCandidate(candidateIDs[transaction.VoteMaximum])
	require.NoError(t, err)
	assert.Equal(t, blockutil.FloatToUint128(t, 333), candidate.VotePower)

	// Reset vote to nil
	bb = bb.
		Tx().Type(dState.TxOpVote).Payload(&transaction.VotePayload{}).SignPair(voter).Execute()

	block = bb.Build()

	voterAcc, err = block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	for _, v := range votePayload.CandidateIDs {
		_, err = voterAcc.Voted.Get(v)
		assert.Error(t, trie.ErrNotFound)
	}

	for _, v := range candidateIDs[:transaction.VoteMaximum-1] {
		candidate, err := block.State().DposState().GetCandidate(v)
		require.NoError(t, err)
		assert.Equal(t, blockutil.FloatToUint128(t, 100000000), candidate.VotePower)
	}

	candidate, err = block.State().DposState().GetCandidate(candidateIDs[transaction.VoteMaximum])
	require.NoError(t, err)
	assert.Equal(t, blockutil.FloatToUint128(t, 0), candidate.VotePower)

}
