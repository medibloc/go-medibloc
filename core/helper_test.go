package core_test

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockID int

var (
	chainID              uint32  = 1010
	genesisID            blockID = -1
	genesisBlock         *core.Block
	blockpoolTestDataDir = "./testdata/blockpool"
)

func init() {
	conf, _ := core.LoadGenesisConf(defaultGenesisConfPath)
	s, _ := storage.NewMemoryStorage()
	genesisBlock, _ = core.NewGenesisBlock(conf, s)
	chainID = conf.Meta.ChainId
}

// newBlockTestSet generates test block set from blockID to parentBlockID index
//
// Example
// idxToParent := []blockID{genesisID, 0, 0, 1, 1, 2, 2}
//
//                     [genesis]
//                         |
//                        [0]
// idxToParent ==>      /     \
//                    [1]     [2]
//                    / \     / \
//                  [3] [4] [5] [6]
func newBlockTestSet(t *testing.T, idxToParent []blockID) (blocks map[blockID]*core.Block) {
	blocks = make(map[blockID]*core.Block)
	for i, parentID := range idxToParent {
		if parentID == genesisID {
			blocks[genesisID] = genesisBlock
		}
		parentBlock, ok := blocks[parentID]
		require.True(t, ok)

		newBlock := newTestBlock(t, parentBlock)
		newBlockID := blockID(i)
		blocks[newBlockID] = newBlock
	}
	return blocks
}

func newTestBlock(t *testing.T, parent *core.Block) *core.Block {
	block, err := core.NewBlock(chainID, RandomAddress(t), parent)
	require.Nil(t, err)
	require.NotNil(t, block)
	require.EqualValues(t, block.ParentHash(), parent.Hash())
	err = block.Seal()
	require.Nil(t, err)
	return block
}

func RandomAddress(t *testing.T) (addr common.Address) {
	nw, err := io.ReadFull(rand.Reader, addr[:])
	require.Nil(t, err)
	require.EqualValues(t, common.AddressLength, nw)
	return addr
}

func getStorage(t *testing.T) storage.Storage {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	return s
}
