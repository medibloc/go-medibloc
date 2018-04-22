package core_test

import (
	"testing"

	"crypto/rand"
	"math/big"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockID int

var (
	chainID      uint32  = 1010
	genesisID    blockID = -1
	genesisBlock *core.Block
	fromAddress  = "02279dcbc360174b4348685e75287a60abc5290497d2e3330b6a1791c4f35bcd20"
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

func getBlock(t *testing.T, parent *core.Block, coinbaseHex string) *core.Block {
	var coinbase common.Address
	if coinbaseHex == "" {
		coinbase = newAddress(t)
	} else {
		coinbase = common.HexToAddress(coinbaseHex)
	}
	block, err := core.NewBlock(chainID, coinbase, parent)
	require.Nil(t, err)
	require.NotNil(t, block)
	require.EqualValues(t, block.ParentHash(), parent.Hash())
	return block
}

func newTestBlock(t *testing.T, parent *core.Block) *core.Block {
	block := getBlock(t, parent, "")
	err := block.Seal()
	require.Nil(t, err)
	return block
}

func newTestBlockWithTxs(t *testing.T, parent *core.Block) *core.Block {
	block := getBlock(t, parent, fromAddress)
	txs := newTestTransactions(t, block)
	block.SetTransactions(txs)
	return block
}

func newTestTransactions(t *testing.T, block *core.Block) core.Transactions {
	ks := keystore.NewKeyStore()
	txDatas := []struct {
		from    common.Address
		to      common.Address
		amount  *util.Uint128
		privHex string
	}{
		{
			common.HexToAddress("03528fa3684218f32c9fd7726a2839cff3ddef49d89bf4904af11bc12335f7c939"),
			mockAddress(t, ks),
			util.NewUint128FromUint(10),
			"bd516113ecb3ad02f3a5bf750b65a545d56835e3d7ef92159dc655ed3745d5c0",
		},
	}

	var err error
	txs := make(core.Transactions, len(txDatas))
	for i, txData := range txDatas {
		acc, err := block.State().GetAccount(txData.from)
		require.NoError(t, err)
		nonce := acc.Nonce()
		txs[i], err = core.NewTransaction(1, txData.from, txData.to, txData.amount, nonce+1, core.TxPayloadBinaryType, []byte("datadata"))
		require.NoError(t, err)
		sig, err := crypto.NewSignature(algorithm.SECP256K1)
		require.NoError(t, err)
		privKey, err := secp256k1.NewPrivateKeyFromHex(txData.privHex)
		assert.NoError(t, err)
		sig.InitSign(privKey)
		assert.NoError(t, txs[i].SignThis(sig))
	}
	require.NoError(t, err)

	return txs
}
func newTransaction(t *testing.T, fromKey signature.PrivateKey, toKey signature.PrivateKey, nonce uint64) *core.Transaction {
	from, err := common.PublicKeyToAddress(fromKey.PublicKey())
	require.Nil(t, err)
	to, err := common.PublicKeyToAddress(toKey.PublicKey())
	require.Nil(t, err)
	tx, err := core.NewTransaction(chainID, from, to, util.Uint128Zero(), nonce, "", nil)
	require.Nil(t, err)
	return tx
}

func newSignedTransaction(t *testing.T, from signature.PrivateKey, to signature.PrivateKey, nonce uint64) *core.Transaction {
	tx := newTransaction(t, from, to, nonce)
	signTx(t, tx, from)
	return tx
}

func newPrivateKey(t *testing.T) signature.PrivateKey {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	require.NoError(t, err)
	return privKey
}

func newAddress(t *testing.T) common.Address {
	key := newPrivateKey(t)
	addr, err := common.PublicKeyToAddress(key.PublicKey())
	require.NoError(t, err)
	return addr
}

func newNonce(t *testing.T) uint64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	require.NoError(t, err)
	return n.Uint64()
}

func newRandomTransaction(t *testing.T) *core.Transaction {
	from := newPrivateKey(t)
	to := newPrivateKey(t)
	tx := newTransaction(t, from, to, newNonce(t))
	return tx
}

func newRandomSignedTransaction(t *testing.T) *core.Transaction {
	from := newPrivateKey(t)
	to := newPrivateKey(t)
	tx := newTransaction(t, from, to, newNonce(t))
	signTx(t, tx, from)
	return tx
}

func signTx(t *testing.T, tx *core.Transaction, key signature.PrivateKey) {
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(key)
	tx.SignThis(sig)
}

func newKeySlice(t *testing.T, n int) []signature.PrivateKey {
	var keys []signature.PrivateKey
	for i := 0; i < n; i++ {
		keys = append(keys, newPrivateKey(t))
	}
	return keys
}

func getStorage(t *testing.T) storage.Storage {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	return s
}

func mockAddress(t *testing.T, ks *keystore.KeyStore) common.Address {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	assert.NoError(t, err)
	acc, err := ks.SetKey(privKey)
	assert.NoError(t, err)
	return acc
}

func newTestTransactionManagers(t *testing.T, n int) (mgrs []*core.TransactionManager, closeFn func()) {
	// New test network
	tm := net.NewMedServiceTestManager(n, 1)
	svc, err := tm.MakeNewTestMedService()
	require.Nil(t, err)
	require.Len(t, svc, n)

	tm.StartMedServices()
	tm.WaitStreamReady()
	tm.WaitRouteTableSync()

	med, err := medlet.New(&medletpb.Config{
		Chain: &medletpb.ChainConfig{
			ChainId: chainID,
		},
	})
	require.Nil(t, err)

	for i := 0; i < n; i++ {
		mgr := core.NewTransactionManager(med, 1024)
		mgr.RegisterInNetwork(svc[i])
		mgr.Start()
		mgrs = append(mgrs, mgr)
	}

	return mgrs, func() {
		for _, mgr := range mgrs {
			mgr.Stop()
		}
		tm.StopMedServices()
	}
}
