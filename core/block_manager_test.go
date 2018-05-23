// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

package core_test

import (
	"math/rand"
	"testing"

	"github.com/medibloc/go-medibloc/core"
	testutil "github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func restoreBlockData(t *testing.T, block *core.Block) *core.BlockData {
	msg, err := block.ToProto()
	require.Nil(t, err)
	blockData := new(core.BlockData)
	blockData.FromProto(msg)
	return blockData
}

func nextBlockData(t *testing.T, parent *core.Block, dynasties testutil.Dynasties) *core.BlockData {
	block := testutil.NewTestBlockWithTxs(t, parent, dynasties[0])
	require.Nil(t, block.State().TransitionDynasty(block.Timestamp()))
	require.Nil(t, block.ExecuteAll())
	require.Nil(t, block.Seal())
	testutil.SignBlock(t, block, dynasties)

	// restore state for simulate network received message
	return restoreBlockData(t, block)
}

func getBlockDataList(t *testing.T, idxToParent []testutil.BlockID, genesis *core.Block, dynasties testutil.Dynasties) []*core.BlockData {
	from := dynasties[0]
	blockMap := make(map[testutil.BlockID]*core.Block)
	blockMap[testutil.GenesisID] = genesis
	blockDatas := make([]*core.BlockData, len(idxToParent))
	for i, parentID := range idxToParent {
		block := testutil.NewTestBlockWithTxs(t, blockMap[parentID], from)
		require.Nil(t, block.State().TransitionDynasty(block.Timestamp()))
		require.Nil(t, block.ExecuteAll())
		require.Nil(t, block.Seal())
		testutil.SignBlock(t, block, dynasties)
		blockMap[testutil.BlockID(i)] = block

		// restore state for simulate network received message
		blockDatas[i] = restoreBlockData(t, block)
	}

	return blockDatas
}

func TestBlockManager_Sequential(t *testing.T) {
	m := testutil.NewMockMedlet(t)
	bm := m.BlockManager()
	genesis := bm.TailBlock()
	dynasties := m.Dynasties()

	idxToParent := []testutil.BlockID{testutil.GenesisID, 0, 1, 2, 3, 4, 5}
	blockMap := make(map[testutil.BlockID]*core.Block)
	blockMap[testutil.GenesisID] = genesis
	for idx, parentID := range idxToParent {
		blockData := nextBlockData(t, blockMap[parentID], dynasties)

		err := bm.PushBlockData(blockData)
		assert.Nil(t, err)
		assert.Equal(t, bm.TailBlock().Hash(), blockData.Hash())
		blockMap[testutil.BlockID(idx)] = bm.TailBlock()
	}
}

func TestBlockManager_Reverse(t *testing.T) {
	m := testutil.NewMockMedlet(t)
	bm := m.BlockManager()
	genesis := bm.TailBlock()
	dynasties := m.Dynasties()

	idxToParent := []testutil.BlockID{testutil.GenesisID, 0, 1, 2, 3, 4, 5}
	blockDatas := getBlockDataList(t, idxToParent, genesis, dynasties)

	for i := len(idxToParent) - 1; i >= 0; i-- {
		blockData := blockDatas[i]
		err := bm.PushBlockData(blockData)
		require.Nil(t, err)
		if i > 0 {
			require.Equal(t, genesis.Hash(), bm.TailBlock().Hash())
		} else {
			assert.Equal(t, blockDatas[len(idxToParent)-1].Hash(), bm.TailBlock().Hash())
		}
	}
}

func TestBlockManager_Tree(t *testing.T) {
	m := testutil.NewMockMedlet(t)
	bm := m.BlockManager()
	genesis := bm.TailBlock()
	dynasties := m.Dynasties()

	tests := []struct {
		idxToParent []testutil.BlockID
	}{
		{[]testutil.BlockID{testutil.GenesisID, 0, 0, 1, 1, 1, 3, 3, 3, 3, 7, 7}},
		{[]testutil.BlockID{testutil.GenesisID, 0, 0, 1, 2, 2, 2, 2, 3, 3, 7, 7}},
		{[]testutil.BlockID{testutil.GenesisID, 0, 0, 1, 2, 3, 3, 2, 5, 5, 6, 7}},
	}

	for _, test := range tests {
		blockDatas := getBlockDataList(t, test.idxToParent, genesis, dynasties)
		for i := range blockDatas {
			j := rand.Intn(i + 1)
			blockDatas[i], blockDatas[j] = blockDatas[j], blockDatas[i]
		}
		for _, blockData := range blockDatas {
			err := bm.PushBlockData(blockData)
			require.Nil(t, err)
		}
	}
}
