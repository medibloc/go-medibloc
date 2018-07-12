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

	addr := candidate.Addr
	key := candidate.PrivKey
	txType := dpos.TxOpBecomeCandidate
	bb = bb.
		Tx().From(addr).Nonce(1).Type(txType).Value(1000000001).CalcHash().SignKey(key).ExecuteErr(core.ErrBalanceNotEnough).
		Tx().From(addr).Nonce(1).Type(txType).Value(10).CalcHash().SignKey(key).Execute().
		Tx().From(addr).Nonce(2).Type(txType).Value(10).CalcHash().SignKey(key).ExecuteErr(dpos.ErrAlreadyCandidate)

	bb.Expect().Balance(addr, uint64(1000000000-10))
	block := bb.Build()

	pbCandidate, err := block.State().DposState().(*dpos.State).Candidate(addr)
	assert.NoError(t, err)

	tenBytes, err := util.NewUint128FromUint(10).ToFixedSizeByteSlice()
	assert.Equal(t, pbCandidate.Collatral, tenBytes)

	tbQuit := blockutil.New(t, testutil.DynastySize).Tx().From(addr).Nonce(2).
		Type(dpos.TxOpQuitCandidacy)

	transactionQuit1 := tbQuit.CalcHash().SignKey(key).Build()
	transactionQuit2 := tbQuit.Nonce(3).CalcHash().SignKey(key).Build()

	bb = bb.ExecuteTx(transactionQuit1).
		ExecuteTxErr(transactionQuit2, dpos.ErrNotCandidate)
	bb.Expect().Balance(addr, uint64(1000000000))
	block = bb.Build()

	_, err = block.State().DposState().(*dpos.State).Candidate(addr)
	assert.Equal(t, dpos.ErrNotCandidate, err)
}

func TestVote(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	candidate := bb.TokenDist[testutil.DynastySize]
	voter := bb.TokenDist[testutil.DynastySize+1]

	bb = bb.
		Tx().From(voter.Addr).Nonce(1).Value(333).Type(core.TxOpVest).CalcHash().SignKey(voter.PrivKey).Execute().
		Tx().From(candidate.Addr).Nonce(1).Value(10).Type(dpos.TxOpBecomeCandidate).CalcHash().SignKey(candidate.PrivKey).Execute().
		Tx().From(voter.Addr).Nonce(2).To(candidate.Addr).Type(dpos.TxOpVote).CalcHash().SignKey(voter.PrivKey).Execute()

	bb.Expect().Balance(candidate.Addr, uint64(1000000000-10))
	block := bb.Build()

	voterAcc, err := block.State().GetAccount(voter.Addr)
	assert.NoError(t, err)
	assert.Equal(t, voterAcc.Voted(), candidate.Addr.Bytes())

	pbCandidate, err := block.State().DposState().(*dpos.State).Candidate(candidate.Addr)
	assert.NoError(t, err)

	expectedVotePower, err := util.NewUint128FromUint(333).ToFixedSizeByteSlice()
	require.NoError(t, err)
	assert.Equal(t, pbCandidate.VotePower, expectedVotePower)

}
