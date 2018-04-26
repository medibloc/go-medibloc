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
	"github.com/medibloc/go-medibloc/util/test"
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

func nextBlockData(t *testing.T, parent *core.Block) *core.BlockData {
	block := test.NewTestBlockWithTxs(t, parent)
	require.Nil(t, block.ExecuteAll())
	require.Nil(t, block.Seal())

	// restore state for simiulate network received message
	return restoreBlockData(t, block)
}

func getBlockDataList(t *testing.T, idxToParent []test.BlockID) []*core.BlockData {
	blockMap := make(map[test.BlockID]*core.Block)
	blockMap[test.GenesisID] = test.GenesisBlock
	blockDatas := make([]*core.BlockData, len(idxToParent))
	for i, parentId := range idxToParent {
		block := test.NewTestBlockWithTxs(t, blockMap[parentId])
		require.Nil(t, block.ExecuteAll())
		require.Nil(t, block.Seal())
		blockMap[test.BlockID(i)] = block

		// restore state for simiulate network received message
		blockDatas[i] = restoreBlockData(t, block)
	}

	return blockDatas
}

func TestBlockSubscriber_Sequential(t *testing.T) {
	m := test.NewMockMedlet(t)
	bm, err := core.NewBlockManager(m.Config())
	require.NoError(t, err)
	bm.Setup(m.Genesis(), m.Storage(), m.NetService(), m.Consensus())

	idxToParent := []test.BlockID{test.GenesisID, 0, 1, 2, 3, 4, 5}
	blockMap := make(map[test.BlockID]*core.Block)
	blockMap[test.GenesisID] = test.GenesisBlock
	for idx, parentId := range idxToParent {
		blockData := nextBlockData(t, blockMap[parentId])

		err = bm.PushBlockData(blockData)
		assert.Nil(t, err)
		assert.Equal(t, bm.TailBlock().Hash(), blockData.Hash())
		blockMap[test.BlockID(idx)] = bm.TailBlock()
	}
}

func TestBlockSubscriber_Reverse(t *testing.T) {
	m := test.NewMockMedlet(t)
	bm, err := core.NewBlockManager(m.Config())
	require.NoError(t, err)
	bm.Setup(m.Genesis(), m.Storage(), m.NetService(), m.Consensus())

	idxToParent := []test.BlockID{test.GenesisID, 0, 1, 2, 3, 4, 5}
	blockDatas := getBlockDataList(t, idxToParent)

	for i := len(idxToParent) - 1; i >= 0; i-- {
		blockData := blockDatas[i]
		err = bm.PushBlockData(blockData)
		require.Nil(t, err)
		if i > 0 {
			require.Equal(t, test.GenesisBlock.Hash(), bm.TailBlock().Hash())
		} else {
			assert.Equal(t, blockDatas[len(idxToParent)-1].Hash(), bm.TailBlock().Hash())
		}
	}
}

func TestBlockSubscriber_Tree(t *testing.T) {
	m := test.NewMockMedlet(t)
	bm, err := core.NewBlockManager(m.Config())
	require.NoError(t, err)
	bm.Setup(m.Genesis(), m.Storage(), m.NetService(), m.Consensus())

	tests := []struct {
		idxToParent []test.BlockID
	}{
		{[]test.BlockID{test.GenesisID, 0, 0, 1, 1, 1, 3, 3, 3, 3, 7, 7}},
		{[]test.BlockID{test.GenesisID, 0, 0, 1, 2, 2, 2, 2, 3, 3, 7, 7}},
		{[]test.BlockID{test.GenesisID, 0, 0, 1, 2, 3, 3, 2, 5, 5, 6, 7}},
	}

	for _, test := range tests {
		blockDatas := getBlockDataList(t, test.idxToParent)
		for i := range blockDatas {
			j := rand.Intn(i + 1)
			blockDatas[i], blockDatas[j] = blockDatas[j], blockDatas[i]
		}
		for _, blockData := range blockDatas {
			err = bm.PushBlockData(blockData)
			require.Nil(t, err)
		}
	}
}
