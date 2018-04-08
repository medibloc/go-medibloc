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

func TestBlockSubscriber_Sequntial(t *testing.T) {
	bc, bp := getBcBp(t)

	idxToParent := []blockID{genesisID, 0, 1, 2, 3, 4, 5}
	blockMap := newBlockTestSet(t, idxToParent)

	for _, idx := range idxToParent[1:] {
		block := blockMap[idx]
		err := core.PushBlock(block, bc, bp)
		assert.Nil(t, err)
		assert.Equal(t, bc.MainTailBlock().Hash(), block.Hash())
		assert.Equal(t, false, bp.Has(block))
	}
}

func TestBlockSubscriber_Reverse(t *testing.T) {
	bc, bp := getBcBp(t)

	idxToParent := []blockID{genesisID, 0, 1, 2, 3, 4, 5}
	blockMap := newBlockTestSet(t, idxToParent)
	assert.Equal(t, blockMap[genesisID].Hash(), bc.MainTailBlock().Hash())

	for i := len(idxToParent) - 1; i > 0; i-- {
		block := blockMap[idxToParent[i]]
		err := core.PushBlock(block, bc, bp)
		assert.Nil(t, err)
		if i != 1 {
			assert.Equal(t, blockMap[genesisID].Hash(), bc.MainTailBlock().Hash())
			assert.Equal(t, true, bp.Has(block))
		} else {
			assert.Equal(t, blockMap[idxToParent[len(idxToParent)-1]].Hash(), bc.MainTailBlock().Hash())
			assert.Equal(t, false, bp.Has(block))
		}
	}
}
