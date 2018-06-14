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

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getBlockChain(t *testing.T) (*core.BlockChain, *testutil.MockMedlet) {
	m := testutil.NewMockMedlet(t)

	bc, err := core.NewBlockChain(m.Config())
	require.NoError(t, err)
	err = bc.Setup(m.Genesis(), m.Consensus(), m.Storage())
	require.NoError(t, err)
	return bc, m

}

func getBlockSlice(blockMap map[testutil.BlockID]*core.Block, id testutil.BlockID) []*core.Block {
	return []*core.Block{blockMap[id]}
}

func TestBlockChain_OnePath(t *testing.T) {
	bc, m := getBlockChain(t)
	genesis := m.BlockManager().TailBlock()

	idxToParent := []testutil.BlockID{testutil.GenesisID, 0, 1, 2, 3, 4, 5}
	blockMap := testutil.NewBlockTestSet(t, genesis, idxToParent)

	err := bc.PutVerifiedNewBlocks(blockMap[0], getBlockSlice(blockMap, 0), getBlockSlice(blockMap, 0))
	assert.NotNil(t, err)

	err = bc.PutVerifiedNewBlocks(genesis, getBlockSlice(blockMap, 0), getBlockSlice(blockMap, 0))
	assert.Nil(t, err)

	for _, idx := range idxToParent[2:] {
		blocks := getBlockSlice(blockMap, idx)
		err = bc.PutVerifiedNewBlocks(blockMap[testutil.BlockID(idx-1)], blocks, blocks)
		assert.Nil(t, err)
		err = bc.SetTailBlock(blocks[0])
		assert.Nil(t, err)
		assert.Equal(t, blocks[0].Hash(), bc.MainTailBlock().Hash())
	}
}

func TestBlockChain_Tree(t *testing.T) {
	tests := []struct {
		tree []testutil.BlockID
	}{
		{[]testutil.BlockID{testutil.GenesisID, 0, 0, 1, 1, 1, 1}},
		{[]testutil.BlockID{testutil.GenesisID, 0, 0, 0, 0, 1, 2, 2, 3}},
		{[]testutil.BlockID{testutil.GenesisID, 0, 1, 1, 2, 2, 3, 3}},
	}
	// Put one-by-one
	for _, test := range tests {
		bc, m := getBlockChain(t)
		genesis := m.BlockManager().TailBlock()
		blockMap := testutil.NewBlockTestSet(t, genesis, test.tree)
		for idx, parentID := range test.tree {
			blocks := getBlockSlice(blockMap, testutil.BlockID(idx))
			err := bc.PutVerifiedNewBlocks(blockMap[parentID], blocks, blocks)
			assert.Nil(t, err)
		}
	}
	// Put all
	for _, test := range tests {
		bc, m := getBlockChain(t)
		genesis := m.BlockManager().TailBlock()
		blockMap := testutil.NewBlockTestSet(t, genesis, test.tree)
		notTail := make(map[testutil.BlockID]bool)
		for _, idx := range test.tree {
			notTail[idx] = true
		}
		allBlocks := make([]*core.Block, 0)
		tailBlocks := make([]*core.Block, 0)
		for id, block := range blockMap {
			allBlocks = append(allBlocks, block)
			if !notTail[id] {
				tailBlocks = append(tailBlocks, block)
			}
		}
		/* TODO handle when tailBlocks are wrong
		err := bc.PutVerifiedNewBlocks(genesisBlock, allBlocks, allBlocks)
		assert.NotNil(t, err)
		*/
		err := bc.PutVerifiedNewBlocks(genesis, allBlocks, tailBlocks)
		assert.Nil(t, err)
	}
}
