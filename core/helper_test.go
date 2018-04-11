package core_test

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
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

func newTestTransaction(t *testing.T, from common.Address, to common.Address, nonce uint64) *core.Transaction {
	tx, err := core.NewTransaction(chainID, from, to, util.Uint128Zero(), nonce, "", nil)
	require.Nil(t, err)
	return tx
}

func RandomAddress(t *testing.T) (addr common.Address) {
	nw, err := io.ReadFull(rand.Reader, addr[:])
	require.Nil(t, err)
	require.EqualValues(t, common.AddressLength, nw)
	return addr

}

func newPrivateKey(t *testing.T) signature.PrivateKey {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	assert.NoError(t, err)
	return privKey
}

func signTx(t *testing.T, tx *core.Transaction) {
	key := newPrivateKey(t)
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(key)
	tx.SignThis(sig)
}

func newTestAddrSet(t *testing.T, n int) (addrs []common.Address) {
	for i := 0; i < n; i++ {
		addrs = append(addrs, RandomAddress(t))
	}
	return addrs
}

func getStorage(t *testing.T) storage.Storage {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	return s
}
