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

	"github.com/medibloc/go-medibloc/util/testutil/blockutil"

	"github.com/stretchr/testify/assert"

	"github.com/medibloc/go-medibloc/util/testutil"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/stretchr/testify/require"
)

func TestNewChainManager(t *testing.T) {
	cfg := medlet.DefaultConfig()
	cfg.Chain.TailCacheSize = 0

	bc, err := core.NewBlockChain(cfg)
	require.NoError(t, err)

	_, err = core.NewChainManager(cfg, bc)
	require.EqualError(t, err, "Must provide a positive size")
}

func TestChainManager_Setup(t *testing.T) {
	network := testutil.NewNetwork(t, testutil.DynastySize)
	seed := network.NewSeedNode()

	cm := seed.Med.ChainManager()
	err := cm.Setup(seed.Med.TransactionManager(), seed.Med.Consensus())
	require.NoError(t, err)

	assert.Equal(t, cm.MainTailBlock(), seed.Tail())
	assert.Equal(t, cm.LIB(), seed.Tail())
}

func TestChainManager_SetLIB(t *testing.T) {
	dynastySize := 6
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()
	cm := seed.Med.ChainManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	prevLIB := cm.LIB()
	LIB := bb.Block(seed.GenesisBlock()).Child().SignProposer().Build()
	err := bm.PushBlockData(LIB.BlockData)
	assert.NoError(t, err)
	err = seed.WaitUntilTailHeight(LIB.Height(), 10*time.Second)

	err = cm.SetLIB(LIB)
	assert.NoError(t, err)
	assert.Equal(t, cm.LIB().Hash(), LIB.Hash())

	err = cm.SetLIB(LIB)
	assert.NoError(t, err)
	assert.Equal(t, cm.LIB().Hash(), LIB.Hash())

	// can not revert lib
	err = cm.SetLIB(prevLIB)
	assert.NoError(t, err)
	assert.Equal(t, cm.LIB().Hash(), LIB.Hash())

	// test removeForkedBranch
	newLIB := bb.Block(LIB).Child().SignProposer().Build()
	err = bm.PushBlockData(newLIB.BlockData)
	assert.NoError(t, err)
	err = seed.WaitUntilTailHeight(newLIB.Height(), 10*time.Second)
	assert.NoError(t, err)

	forkedBlock := bb.Block(LIB).Child().Tx().RandomTx().Execute().SignProposer().Build()
	err = bm.PushBlockDataSync(forkedBlock.BlockData, 10*time.Second)
	assert.NoError(t, err)
	err = seed.WaitUntilBlockAcceptedOnChain(forkedBlock.Hash(), 10*time.Second)
	assert.NoError(t, err)
	blockFromChain := cm.BlockByHash(forkedBlock.Hash())
	assert.NotNil(t, blockFromChain)

	err = cm.SetLIB(newLIB)
	assert.NoError(t, err)
	assert.Equal(t, cm.LIB().Hash(), newLIB.Hash())

	blockFromChain = cm.BlockByHash(forkedBlock.Hash())
	assert.Nil(t, blockFromChain)
}

func TestChainManager_TailBlocks(t *testing.T) {
	blocks := 5
	dynastySize := 6
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()
	cm := seed.Med.ChainManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	payer := bb.KeyPairs[0]
	tail := seed.Tail()
	for i := 0; i < blocks; i++ {
		newTail := bb.Block(tail).Child().Tx().Nonce(5).Type(core.TxOpTransfer).Value(float64(i + 1)).To(payer.
			Addr).
			SignPair(payer).Execute().SignProposer().Build()
		err := bm.PushBlockDataSync(newTail.BlockData, 10*time.Second)
		assert.NoError(t, err)
		err = seed.WaitUntilBlockAcceptedOnChain(newTail.Hash(), 10*time.Second)
		assert.NoError(t, err)
	}
	assert.Equal(t, blocks, len(cm.TailBlocks()))
}
