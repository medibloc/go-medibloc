package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"crypto/rand"
	"io"
)

var (
	chainID   uint32 = 1010
	genesisID        = 0
	coinbase         = common.Address{}
)

func randomAddress(t *testing.T) (addr common.Address) {
	nw, err := io.ReadFull(rand.Reader, addr[:])
	require.Nil(t, err)
	require.EqualValues(t, common.AddressLength, nw)
	return addr
}

func generateBlocks(t *testing.T, parentIndex []int) (blocks []*core.Block) {
	blocks = make([]*core.Block, len(parentIndex))
	for i, parentID := range parentIndex {
		parentBlock := core.TODOTestGenesisBlock
		if parentID != genesisID {
			require.True(t, parentID < i)
			parentBlock = blocks[parentID]
		}

		blocks[i] = generateChildBlock(t, parentBlock)
	}
	blocks = append(blocks, core.TODOTestGenesisBlock)
	return blocks
}

func generateChildBlock(t *testing.T, parent *core.Block) *core.Block {
	block, err := core.NewBlock(chainID, randomAddress(t), parent)
	require.Nil(t, err)
	require.EqualValues(t, block.ParentHash(), parent.Hash())
	err = block.Seal()
	require.Nil(t, err)
	return block
}

func TestBlockPoolPush(t *testing.T) {
	tests := []struct {
		name        string
		parentIndex []int
	}{
		{"case 1", []int{0, 0, 1, 2, 1, 3, 1}},
		{"case 2", []int{0, 0, 1, 1, 1, 1, 1}},
		{"case 3", []int{0, 0, 1, 2, 3, 4, 5}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bp, err := core.NewBlockPool(128)
			assert.Nil(t, err)

			blocks := generateBlocks(t, test.parentIndex)
			for _, block := range blocks {
				err = bp.Push(block)
				assert.Nil(t, err)
			}
			for i, parent := range test.parentIndex {
				// Check parent block
				if test.parentIndex[i] == genesisID {
					continue
				}
				expected := blocks[parent]
				actual := bp.GetParentBlock(blocks[i])
				assert.Equal(t, expected, actual)

				// Check children blocks
				childBlocks := make([]*core.Block, 0)
				for j, parent := range test.parentIndex {
					if parent == i {
						childBlocks = append(childBlocks, blocks[j])
					}
				}
				if len(childBlocks) == 0 {
					continue
				}
				assert.EqualValues(t, childBlocks, bp.GetChildBlocks(blocks[i]))

				// Check ancestor block
				assert.EqualValues(t, core.TODOTestGenesisBlock, bp.GetAncestorBlock(blocks[i]))
			}
		})
	}
}
