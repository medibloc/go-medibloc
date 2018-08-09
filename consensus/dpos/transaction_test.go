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
		Tx().Type(txType).Value(1000000001).Nonce(1).SignPair(candidate).ExecuteErr(core.ErrBalanceNotEnough).
		Tx().Type(txType).Value(10).Nonce(1).SignPair(candidate).Execute().
		Tx().Type(txType).Value(10).Nonce(2).SignPair(candidate).ExecuteErr(dpos.ErrAlreadyCandidate)

	bb.Expect().Balance(candidate.Addr, uint64(1000000000-10))
	block := bb.Build()

	cs := block.State().DposState().CandidateState()
	as := block.State().AccState()

	_, err := cs.Get(candidate.Addr.Bytes())
	assert.NoError(t, err)

	acc, err := as.GetAccount(candidate.Addr)
	require.NoError(t, err)

	assert.Equal(t, util.NewUint128FromUint(10), acc.Collateral)

	bb = bb.
		Tx().Type(dpos.TxOpQuitCandidacy).Nonce(2).SignPair(candidate).Execute().
		Tx().Type(dpos.TxOpQuitCandidacy).Nonce(3).SignPair(candidate).ExecuteErr(dpos.ErrNotCandidate)

	block = bb.Build()
	as = block.State().AccState()
	cs = block.State().DposState().CandidateState()

	acc, err = as.GetAccount(candidate.Addr)
	require.NoError(t, err)

	assert.Equal(t, util.NewUint128FromUint(0), acc.Collateral)
	assert.Equal(t, 0, len(acc.VotersSlice()))
	assert.Equal(t, util.NewUint128FromUint(0), acc.VotePower)

	_, err = cs.Get(candidate.Addr.Bytes())
	assert.Equal(t, trie.ErrNotFound, err)
}

func TestVote(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	candidate := bb.TokenDist[testutil.DynastySize]
	voter := bb.TokenDist[testutil.DynastySize+1]

	bb = bb.
		Tx().Type(core.TxOpVest).Value(333).Nonce(1).SignPair(voter).Execute().
		Tx().Type(dpos.TxOpBecomeCandidate).Value(10).Nonce(1).SignPair(candidate).Execute().
		Tx().Type(dpos.TxOpVote).To(candidate.Addr).Nonce(2).SignPair(voter).Execute()

	bb.Expect().Balance(candidate.Addr, uint64(1000000000-10))
	block := bb.Build()

	voterAcc, err := block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	assert.Equal(t, voterAcc.VotedSlice()[0], candidate.Addr.Bytes())

	_, err = block.State().DposState().CandidateState().Get(candidate.Addr.Bytes())
	assert.NoError(t, err)

	acc, err := block.State().GetAccount(candidate.Addr)
	require.NoError(t, err)

	assert.Equal(t, util.NewUint128FromUint(333), acc.VotePower)

}
