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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getBcBp(t *testing.T) (*core.BlockChain, *core.BlockPool) {
	bc := getBlockChain(t)
	bp, err := core.NewBlockPool(16)
	require.Nil(t, err)
	return bc, bp
}

func restoreBlockData(t *testing.T, block *core.Block) *core.BlockData {
	msg, err := block.ToProto()
	require.Nil(t, err)
	blockData := new(core.BlockData)
	blockData.FromProto(msg)
	return blockData
}

func nextBlockData(t *testing.T, parent *core.Block) *core.BlockData {
	block := newTestBlockWithTxs(t, parent)
	require.Nil(t, block.ExecuteAll())
	require.Nil(t, block.Seal())

	// restore state for simiulate network received message
	return restoreBlockData(t, block)
}

func getBlockDataList(t *testing.T, idxToParent []blockID) []*core.BlockData {
	blockMap := make(map[blockID]*core.Block)
	blockMap[genesisID] = genesisBlock
	blockDatas := make([]*core.BlockData, len(idxToParent))
	for i, parentId := range idxToParent {
		block := newTestBlockWithTxs(t, blockMap[parentId])
		require.Nil(t, block.ExecuteAll())
		require.Nil(t, block.Seal())
		blockMap[blockID(i)] = block

		// restore state for simiulate network received message
		blockDatas[i] = restoreBlockData(t, block)
	}

	return blockDatas
}

func TestBlockSubscriber_Sequential(t *testing.T) {
	bc, bp := getBcBp(t)
	subscriber := core.NewBlockSubscriber(bp, bc)

	idxToParent := []blockID{genesisID, 0, 1, 2, 3, 4, 5}
	blockMap := make(map[blockID]*core.Block)
	blockMap[genesisID] = genesisBlock
	for idx, parentId := range idxToParent {
		blockData := nextBlockData(t, blockMap[parentId])

		err := subscriber.GetBlockManager().HandleReceivedBlock(blockData, nil)
		assert.Nil(t, err)
		assert.Equal(t, bc.MainTailBlock().Hash(), blockData.Hash())
		assert.Equal(t, false, bp.Has(blockData))
		blockMap[blockID(idx)] = bc.MainTailBlock()
	}
}

func TestBlockSubscriber_Reverse(t *testing.T) {
	bc, bp := getBcBp(t)
	subscriber := core.NewBlockSubscriber(bp, bc)

	idxToParent := []blockID{genesisID, 0, 1, 2, 3, 4, 5}
	blockDatas := getBlockDataList(t, idxToParent)

	for i := len(idxToParent) - 1; i >= 0; i-- {
		blockData := blockDatas[i]
		err := subscriber.GetBlockManager().HandleReceivedBlock(blockData, nil)
		require.Nil(t, err)
		if i > 0 {
			require.Equal(t, genesisBlock.Hash(), bc.MainTailBlock().Hash())
			assert.Equal(t, true, bp.Has(blockData))
		} else {
			assert.Equal(t, blockDatas[len(idxToParent)-1].Hash(), bc.MainTailBlock().Hash())
			assert.Equal(t, false, bp.Has(blockData))
		}
	}
}

func TestBlockSubscriber_Tree(t *testing.T) {
	bc, bp := getBcBp(t)
	subscriber := core.NewBlockSubscriber(bp, bc)

	tests := []struct {
		idxToParent []blockID
	}{
		{[]blockID{genesisID, 0, 0, 1, 1, 1, 3, 3, 3, 3, 7, 7}},
		{[]blockID{genesisID, 0, 0, 1, 2, 2, 2, 2, 3, 3, 7, 7}},
		{[]blockID{genesisID, 0, 0, 1, 2, 3, 3, 2, 5, 5, 6, 7}},
	}

	for _, test := range tests {
		blockDatas := getBlockDataList(t, test.idxToParent)
		for i := range blockDatas {
			j := rand.Intn(i + 1)
			blockDatas[i], blockDatas[j] = blockDatas[j], blockDatas[i]
		}
		for _, blockData := range blockDatas {
			err := subscriber.GetBlockManager().HandleReceivedBlock(blockData, nil)
			require.Nil(t, err)
		}
	}
}
