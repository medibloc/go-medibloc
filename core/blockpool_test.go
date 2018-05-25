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
	testUtil "github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockPoolPush(t *testing.T) {
	tests := []struct {
		name        string
		idxToParent []testUtil.BlockID
	}{
		{"case 1", []testUtil.BlockID{testUtil.GenesisID, 0, 0, 1, 1, 2, 2}},
		{"case 2", []testUtil.BlockID{testUtil.GenesisID, 0, 1, 2, 3, 4, 5}},
		{"case 3", []testUtil.BlockID{testUtil.GenesisID, 0, 0, 0, 0, 0, 0}},
	}
	genesis, _, _ := testUtil.NewTestGenesisBlock(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bp, err := core.NewBlockPool(128)
			assert.Nil(t, err)

			blocks := testUtil.NewBlockTestSet(t, genesis, test.idxToParent)
			for _, block := range blocks {
				err = bp.Push(block)
				assert.Nil(t, err)
			}

			for i, parentID := range test.idxToParent {
				id := testUtil.BlockID(i)
				block := blocks[id]

				// Check finding parent block
				expected := blocks[parentID]
				actual := bp.FindParent(block)
				assert.Equal(t, expected, actual)

				// Check finding children blocks
				childIDs := findChildIDs(test.idxToParent, id)
				childBlocks := make([]core.HashableBlock, 0)
				for _, cid := range childIDs {
					childBlocks = append(childBlocks, blocks[cid])
				}
				assert.True(t, equalBlocks(childBlocks, bp.FindChildren(block)))

				// Check ancestor block
				assert.Equal(t, genesis, bp.FindUnlinkedAncestor(block))
			}
		})
	}
}

func TestBlockPoolEvict(t *testing.T) {
	cacheSize := 3
	bp, err := core.NewBlockPool(cacheSize)
	require.Nil(t, err)

	genesis, _, _ := testUtil.NewTestGenesisBlock(t)

	idxToParent := []testUtil.BlockID{testUtil.GenesisID, 0, 1, 2, 3, 4, 5}
	blockMap := testUtil.NewBlockTestSet(t, genesis, idxToParent)

	blocks := mapToSlice(blockMap)
	for i, block := range blocks {
		err = bp.Push(block)
		assert.Nil(t, err)

		if i >= cacheSize {
			assert.False(t, bp.Has(blocks[i-cacheSize]))
		}
	}
}

func TestWrongCacheSize(t *testing.T) {
	_, err := core.NewBlockPool(-1)
	require.NotNil(t, err)
}

func TestDuplicatedBlock(t *testing.T) {
	bp, err := core.NewBlockPool(128)
	require.Nil(t, err)

	genesis, _, _ := testUtil.NewTestGenesisBlock(t)

	block := testUtil.NewTestBlock(t, genesis)
	err = bp.Push(block)
	assert.Nil(t, err)

	err = bp.Push(block)
	assert.Equal(t, core.ErrDuplicatedBlock, err)
}

func TestWrongPushArgument(t *testing.T) {
	bp, err := core.NewBlockPool(128)
	require.Nil(t, err)

	err = bp.Push(nil)
	assert.Equal(t, core.ErrNilArgument, err)
}

func TestRemove(t *testing.T) {
	bp, err := core.NewBlockPool(128)
	require.Nil(t, err)

	genesis, _, _ := testUtil.NewTestGenesisBlock(t)

	// Push genesis
	err = bp.Push(genesis)
	assert.Nil(t, err)

	// Push genesis's child
	block := testUtil.NewTestBlock(t, genesis)
	err = bp.Push(block)
	assert.Nil(t, err)

	// Check genesis's child
	blocks := bp.FindChildren(genesis)
	assert.Len(t, blocks, 1)
	assert.Equal(t, block, blocks[0])

	// Check block's parent
	parent := bp.FindParent(block)
	assert.Equal(t, genesis, parent)

	// Remove block
	bp.Remove(block)

	// Check again
	assert.False(t, bp.Has(block))
	blocks = bp.FindChildren(genesis)
	assert.Len(t, blocks, 0)
}

func TestNotFound(t *testing.T) {
	bp, err := core.NewBlockPool(128)
	require.Nil(t, err)

	genesis, _, _ := testUtil.NewTestGenesisBlock(t)

	blocks := bp.FindChildren(genesis)
	assert.Len(t, blocks, 0)

	block := bp.FindUnlinkedAncestor(genesis)
	assert.Equal(t, genesis, block)

	block = bp.FindParent(block)
	assert.Nil(t, err)
}

func TestFindBlockWithoutPush(t *testing.T) {
	bp, err := core.NewBlockPool(128)
	require.Nil(t, err)

	genesis, _, _ := testUtil.NewTestGenesisBlock(t)

	grandParent := testUtil.NewTestBlock(t, genesis)
	parent := testUtil.NewTestBlock(t, grandParent)
	child1 := testUtil.NewTestBlock(t, parent)
	child2 := testUtil.NewTestBlock(t, parent)

	err = bp.Push(genesis)
	assert.Nil(t, err)
	err = bp.Push(grandParent)
	assert.Nil(t, err)
	err = bp.Push(child1)
	assert.Nil(t, err)
	err = bp.Push(child2)
	assert.Nil(t, err)

	assert.False(t, bp.Has(parent))

	block := bp.FindParent(parent)
	assert.Equal(t, grandParent, block)

	block = bp.FindUnlinkedAncestor(parent)
	assert.Equal(t, genesis, block)

	blocks := bp.FindChildren(parent)
	assert.Len(t, blocks, 2)
	assert.True(t, equalBlocks([]core.HashableBlock{child1, child2}, blocks))
}

func findChildIDs(idxToParent []testUtil.BlockID, id testUtil.BlockID) (childIDs []testUtil.BlockID) {
	for i, parentID := range idxToParent {
		if parentID == id {
			childIDs = append(childIDs, testUtil.BlockID(i))
		}
	}
	return childIDs
}

func equalBlocks(expected, actual []core.HashableBlock) bool {
	if len(expected) != len(actual) {
		return false
	}
	for _, b1 := range expected {
		found := false
		for _, b2 := range actual {
			if b1 == b2 {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func mapToSlice(blocks map[testUtil.BlockID]*core.Block) (slice []*core.Block) {
	for _, block := range blocks {
		slice = append(slice, block)
	}
	return slice
}
