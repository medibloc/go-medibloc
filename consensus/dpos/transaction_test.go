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
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBecomeAndQuitCandidate(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	candidate := bb.TokenDist[testutil.DynastySize]

	txType := dpos.TxOpBecomeCandidate
	bb = bb.
		Tx().StakeTx(candidate, 10000).Execute().
		Tx().Type(txType).Value(1000000001).Nonce(2).SignPair(candidate).ExecuteErr(core.ErrBalanceNotEnough).
		Tx().Type(txType).Value(10).Nonce(2).SignPair(candidate).Execute().
		Tx().Type(txType).Value(10).Nonce(3).SignPair(candidate).ExecuteErr(dpos.ErrAlreadyCandidate)

	bb.Expect().
		Balance(candidate.Addr, 1000000000-10-10000).
		Vesting(candidate.Addr, 10000)

	block := bb.Build()

	ds := block.State().DposState()
	as := block.State().AccState()

	isCandidate, err := ds.IsCandidate(candidate.Addr)
	assert.NoError(t, err)
	assert.Equal(t, true, isCandidate)

	acc, err := as.GetAccount(candidate.Addr)
	require.NoError(t, err)

	assert.Equal(t, util.NewUint128FromUint(10), acc.Collateral)

	bb = bb.
		Tx().Type(dpos.TxOpQuitCandidacy).Nonce(3).SignPair(candidate).Execute().
		Tx().Type(dpos.TxOpQuitCandidacy).Nonce(4).SignPair(candidate).ExecuteErr(dpos.ErrNotCandidate)

	block = bb.Build()
	as = block.State().AccState()
	ds = block.State().DposState()

	acc, err = as.GetAccount(candidate.Addr)
	require.NoError(t, err)

	assert.Equal(t, util.NewUint128FromUint(0), acc.Collateral)
	assert.Equal(t, 0, len(acc.VotersSlice()))
	assert.Equal(t, util.NewUint128FromUint(0), acc.VotePower)

	isCandidate, err = ds.IsCandidate(candidate.Addr)
	assert.NoError(t, err)
	assert.Equal(t, false, isCandidate)
}

func TestVote(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	candidate := bb.TokenDist[testutil.DynastySize]
	voter := bb.TokenDist[testutil.DynastySize+1]

	votePayload := new(dpos.VotePayload)
	overSizePayload := new(dpos.VotePayload)
	duplicatePayload := new(dpos.VotePayload)
	candidates := append(bb.TokenDist[1:testutil.DynastySize], candidate)
	for _, v := range candidates {
		votePayload.Candidates = append(votePayload.Candidates, v.Addr)
		overSizePayload.Candidates = append(overSizePayload.Candidates, v.Addr)
		duplicatePayload.Candidates = append(duplicatePayload.Candidates, v.Addr)
	}
	overSizePayload.Candidates = append(overSizePayload.Candidates, bb.TokenDist[0].Addr)
	duplicatePayload.Candidates[0] = candidate.Addr

	bb = bb.
		Tx().Type(core.TxOpVest).Value(333).SignPair(voter).Execute().
		Tx().StakeTx(candidate, 10000).Execute().
		Tx().Type(dpos.TxOpBecomeCandidate).Value(10).SignPair(candidate).Execute().
		Tx().Type(dpos.TxOpVote).Payload(overSizePayload).SignPair(voter).ExecuteErr(dpos.ErrOverMaxVote).
		Tx().Type(dpos.TxOpVote).Payload(duplicatePayload).SignPair(voter).ExecuteErr(dpos.ErrDuplicateVote).
		Tx().Type(dpos.TxOpVote).Payload(votePayload).SignPair(voter).Execute()

	bb.Expect().Balance(candidate.Addr, uint64(1000000000-10-10000))
	block := bb.Build()

	isCandidate, err := block.State().DposState().IsCandidate(candidate.Addr)
	assert.NoError(t, err)
	assert.Equal(t, true, isCandidate)

	voterAcc, err := block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	for _, v := range candidates {
		_, err = voterAcc.Voted.Get(v.Addr.Bytes())
		assert.NoError(t, err)
	}

	for _, v := range candidates {
		acc, err := block.State().GetAccount(v.Addr)
		require.NoError(t, err)
		assert.Equal(t, util.NewUint128FromUint(333), acc.VotePower)
		_, err = acc.Voters.Get(voter.Addr.Bytes())
		assert.NoError(t, err)
	}

	// Reset vote to nil
	bb = bb.
		Tx().Type(dpos.TxOpVote).Payload(&dpos.VotePayload{}).SignPair(voter).Execute()

	block = bb.Build()

	voterAcc, err = block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	for _, v := range candidates {
		_, err = voterAcc.Voted.Get(v.Addr.Bytes())
		assert.Equal(t, trie.ErrNotFound, err)
	}

	for _, v := range candidates {
		acc, err := block.State().GetAccount(v.Addr)
		require.NoError(t, err)
		assert.Equal(t, util.NewUint128FromUint(0), acc.VotePower)
		_, err = acc.Voters.Get(voter.Addr.Bytes())
		assert.Equal(t, trie.ErrNotFound, err)
		assert.Equal(t, []byte(nil), acc.Voters.RootHash())
	}

}
