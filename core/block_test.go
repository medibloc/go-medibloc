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
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProposer(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis().Child()

	proposer, err := bb.Build().Proposer()
	require.Error(t, err)

	singingProposer := bb.FindProposer()
	b := bb.SignPair(singingProposer).Build()

	proposer, err = b.Proposer()
	require.NoError(t, err)
	assert.Equal(t, singingProposer.Addr, proposer)
}

func TestBlock_BasicTx(t *testing.T) {
	nt := testutil.NewNetwork(t)
	defer nt.Cleanup()

	seed := nt.NewSeedNode()
	nt.SetProposerFromDynasties(seed)
	seed.Start()

	nt.WaitForEstablished()

	tb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist).Block(seed.GenesisBlock()).Tx()
	tx1 := tb.RandomTx().Build()
	tx2 := tb.Nonce(tx1.Nonce() + 1).RandomTx().Build()
	tx3 := tb.Nonce(tx2.Nonce() + 1).RandomTx().Build()

	failed := seed.Med.TransactionManager().PushAndExclusiveBroadcast(tx1, tx2, tx3)
	require.Zero(t, len(failed))

	for seed.Tail().Height() < 2 {
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBlock_PayReward(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis()
	parent := bb.Build()

	bb = bb.Child().Stake().Tx().RandomTx().Execute().Flush()
	proposer := bb.FindProposer()

	cons := dpos.New(blockutil.DynastySize)

	// wrong reward value (calculate reward based on wrong supply)
	block := bb.Clone().Supply("1234567890000000000000").Coinbase(proposer.Addr).PayReward().Seal().CalcHash().SignKey(proposer.PrivKey).Build()
	_, err := parent.CreateChildWithBlockData(block.GetBlockData(), cons)
	assert.Equal(t, core.ErrInvalidBlockReward, err)

	// wrong supply on header
	block = bb.Clone().Coinbase(proposer.Addr).PayReward().Seal().Supply("1234567890000000000000").CalcHash().SignKey(proposer.PrivKey).Build()
	_, err = parent.CreateChildWithBlockData(block.GetBlockData(), cons)
	assert.Equal(t, core.ErrInvalidBlockSupply, err)

}
