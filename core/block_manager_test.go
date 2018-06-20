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
	"math/rand"
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/util/testutil"
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
	tm := m.TransactionManager()
	bm.InjectTransactionManager(tm)
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

func TestBlockManager_CircularParentLink(t *testing.T) {
	m := testutil.NewMockMedlet(t)
	bm := m.BlockManager()
	genesis := bm.TailBlock()
	dynasties := m.Dynasties()

	block := testutil.NewTestBlock(t, genesis)
	testutil.SignBlock(t, block, dynasties)
	err := bm.PushBlockData(block.GetBlockData())
	require.NoError(t, err)

	parent := block.ParentHash()
	hash := block.Hash()
	pb := &corepb.Block{
		Header: &corepb.BlockHeader{
			Hash:       parent,
			ParentHash: hash,
			ChainId:    1,
		},
	}
	var bd core.BlockData
	err = bd.FromProto(pb)
	require.NoError(t, err)
	err = bm.PushBlockData(&bd)
	require.Error(t, core.ErrInvalidBlockHash)
}

func TestBlockManager_FilterByLIB(t *testing.T) {
	m := testutil.NewMockMedlet(t)
	bm := m.BlockManager()
	genesis := bm.TailBlock()
	dynasties := m.Dynasties()
	dynastySize := int(m.Genesis().GetMeta().GetDynastySize())

	idxToParent := []testutil.BlockID{testutil.GenesisID}
	for i := 0; i < dynastySize; i++ {
		idxToParent = append(idxToParent, testutil.BlockID(i))
	}
	idxToParent = append(idxToParent, testutil.BlockID(0))
	idxToParent = append(idxToParent, testutil.BlockID(1))
	idxToParent = append(idxToParent, testutil.BlockID(dynastySize*2/3))
	idxToParent = append(idxToParent, testutil.BlockID(dynastySize*2/3+1))

	blockDatas := getBlockDataList(t, idxToParent, genesis, dynasties)
	for i := 0; i < dynastySize; i++ {
		err := bm.PushBlockData(blockDatas[i])
		assert.NoError(t, err)
	}

	assert.Len(t, blockDatas, dynastySize+5)
	err := bm.PushBlockData(blockDatas[dynastySize+1])
	assert.Equal(t, core.ErrCannotRevertLIB, err)
	err = bm.PushBlockData(blockDatas[dynastySize+2])
	assert.Equal(t, core.ErrCannotRevertLIB, err)
	err = bm.PushBlockData(blockDatas[dynastySize+3])
	assert.NoError(t, err)
	err = bm.PushBlockData(blockDatas[dynastySize+4])
	assert.NoError(t, err)

	parent, err := bm.BlockByHeight(3)
	require.Nil(t, err)
	bd := nextBlockData(t, parent, dynasties)
	bd.SetHeight(20)
	err = bm.PushBlockData(bd)
	assert.Equal(t, core.ErrCannotRevertLIB, err)
}

func TestBlockManager_PruneByLIB(t *testing.T) {
	m := testutil.NewMockMedlet(t)
	bm := m.BlockManager()
	tm := m.TransactionManager()
	bm.InjectTransactionManager(tm)
	genesis := bm.TailBlock()
	dynasties := m.Dynasties()
	dynastySize := int(m.Genesis().GetMeta().GetDynastySize())

	idxToParent := []testutil.BlockID{testutil.GenesisID, 0, 0}
	for i := 1; i < dynastySize; i++ {
		idxToParent = append(idxToParent, testutil.BlockID(i+1))
	}

	blockDatas := getBlockDataList(t, idxToParent, genesis, dynasties)
	for _, blockData := range blockDatas {
		err := bm.PushBlockData(blockData)
		assert.NoError(t, err)
	}

	assert.Nil(t, bm.BlockByHash(blockDatas[1].Hash()))
	assert.NotNil(t, bm.BlockByHash(blockDatas[2].Hash()))
}

func TestBlockManager_InvalidHeight(t *testing.T) {
	m := testutil.NewMockMedlet(t)
	bm := m.BlockManager()
	genesis := bm.TailBlock()
	dynasties := m.Dynasties()

	idxToParent := []testutil.BlockID{testutil.GenesisID, 0, 1, 2, 3, 4, 5}
	blockDatas := getBlockDataList(t, idxToParent, genesis, dynasties)
	for _, blockData := range blockDatas {
		err := bm.PushBlockData(blockData)
		assert.NoError(t, err)
	}

	parent, err := bm.BlockByHeight(3)
	require.Nil(t, err)
	bd := nextBlockData(t, parent, dynasties)
	tests := []struct {
		height uint64
		err    error
	}{
		{0, core.ErrCannotRevertLIB},
		{1, core.ErrCannotRevertLIB},
		{2, core.ErrCannotExecuteOnParentBlock},
		{3, core.ErrCannotExecuteOnParentBlock},
		{5, core.ErrCannotExecuteOnParentBlock},
		{6, core.ErrCannotExecuteOnParentBlock},
		{999, core.ErrCannotExecuteOnParentBlock},
		{4, nil},
	}
	for _, test := range tests {
		bd.SetHeight(test.height)
		assert.Equal(t, test.err, bm.PushBlockData(bd), "testcase = %v", test)
	}
}
