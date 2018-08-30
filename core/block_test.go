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

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMiner(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	proposer, err := bb.Build().Proposer()
	require.Error(t, err)

	singingMiner := bb.FindMiner()
	b := bb.SignPair(singingMiner).Build()

	proposer, err = b.Proposer()
	require.NoError(t, err)
	assert.Equal(t, singingMiner.Addr, proposer)
}

func TestBlock_BasicTx(t *testing.T) {
	nt := testutil.NewNetwork(t, 3)
	defer nt.Cleanup()
	nt.SetLogTestHook()

	seed := nt.NewSeedNode()
	nt.SetMinerFromDynasties(seed)
	seed.Start()

	nt.WaitForEstablished()

	bb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist)
	tx1 := bb.Block(seed.GenesisBlock()).Tx().RandomTx().Build()
	tx2 := bb.Block(seed.GenesisBlock()).Tx().RandomTx().Build()
	tx3 := bb.Block(seed.GenesisBlock()).Tx().RandomTx().Build()

	err := seed.Med.TransactionManager().Push(tx1)
	require.NoError(t, err)
	err = seed.Med.TransactionManager().Push(tx2)
	require.NoError(t, err)
	err = seed.Med.TransactionManager().Push(tx3)
	require.NoError(t, err)

	for seed.Tail().Height() < 2 {
		time.Sleep(100 * time.Millisecond)
	}
}

func TestBlock_PayReward(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()
	parent := bb.Build()

	bb = bb.Child().Stake().Tx().RandomTx().Execute()
	miner := bb.FindMiner()

	// wrong reward value (calculate reward based on wrong supply)
	block := bb.Supply("1234567890000000000000").PayReward().Seal().CalcHash().SignKey(miner.PrivKey).Build()
	_, err := block.GetBlockData().ExecuteOnParentBlock(parent, blockutil.DefaultTxMap)
	assert.Equal(t, core.ErrInvalidBlockReward, err)

	// wrong supply on header
	block = bb.PayReward().Seal().Supply("1234567890000000000000").CalcHash().SignKey(miner.PrivKey).Build()
	_, err = block.GetBlockData().ExecuteOnParentBlock(parent, blockutil.DefaultTxMap)
	assert.Equal(t, core.ErrInvalidBlockSupply, err)

}
