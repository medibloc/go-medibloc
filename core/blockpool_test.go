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
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlockPool(t *testing.T) {
	_, err := core.NewBlockPool(-1)
	require.NotNil(t, err)
}

func TestBlockPoolEvict(t *testing.T) {
	var (
		cacheSize = 3
		nBlocks   = 5
		blocks    []*core.Block
	)
	bp, err := core.NewBlockPool(cacheSize)
	require.Nil(t, err)

	bb := blockutil.New(t, blockutil.DynastySize).Genesis()

	tail := bb.Build()
	for i := 0; i < nBlocks; i++ {
		mint := bb.Block(tail).Child().SignProposer().Build()
		blocks = append(blocks, mint)
		tail = mint
	}

	for i, block := range blocks {
		err = bp.Push(block)
		assert.Nil(t, err)

		if i >= cacheSize {
			assert.False(t, bp.Has(blocks[i-cacheSize]))
		}
	}
}

func TestDuplicatedBlock(t *testing.T) {
	bp, err := core.NewBlockPool(128)
	require.Nil(t, err)

	block := blockutil.New(t, blockutil.DynastySize).Genesis().Child().SignProposer().Build()

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

	bb := blockutil.New(t, blockutil.DynastySize).Genesis()
	genesis := bb.Build()

	// Push genesis
	err = bp.Push(genesis)
	assert.Nil(t, err)

	// Push genesis's child
	block := bb.Child().SignProposer().Build()
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

	genesis := blockutil.New(t, blockutil.DynastySize).Genesis().Build()

	blocks := bp.FindChildren(genesis)
	assert.Len(t, blocks, 0)

	block := bp.FindUnlinkedAncestor(genesis)
	assert.Equal(t, nil, block)

	block = bp.FindParent(genesis)
	assert.Nil(t, err)
}

func TestFindBlockWithoutPush(t *testing.T) {
	bp, err := core.NewBlockPool(128)
	require.Nil(t, err)

	bb := blockutil.New(t, blockutil.DynastySize).Genesis()
	genesis := bb.Build()

	bb = bb.Child().Stake().SignProposer()
	grandParent := bb.Build()
	bb = bb.Child().SignProposer()
	parent := bb.Build()
	child1 := bb.Child().Tx().RandomTx().Execute().SignProposer().Build()
	child2 := bb.Child().Tx().RandomTx().Execute().SignProposer().Build()

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
