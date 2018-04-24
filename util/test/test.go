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

package test

import (
	"os"
	"path"
	"testing"

	"crypto/rand"
	"math/big"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
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

// BlockID block ID
type BlockID int

var (
	// ChainID chain ID
	ChainID uint32 = 1010
	// DefaultGenesisConfPath default genesis file path
	DefaultGenesisConfPath = path.Join(os.Getenv("GOPATH"), "src/github.com/medibloc/go-medibloc", "conf/default/genesis.conf")
	// GenesisID genesis ID
	GenesisID BlockID = -1
	// GenesisBlock genesis Block
	GenesisBlock *core.Block

	fromAddress = "02279dcbc360174b4348685e75287a60abc5290497d2e3330b6a1791c4f35bcd20"
)

func init() {
	conf, _ := core.LoadGenesisConf(DefaultGenesisConfPath)
	s, _ := storage.NewMemoryStorage()
	GenesisBlock, _ = core.NewGenesisBlock(conf, s)
	ChainID = conf.Meta.ChainId
}

// NewBlockTestSet generates test block set from BlockID to parentBlockID index
//
// Example
// idxToParent := []BlockID{genesisID, 0, 0, 1, 1, 2, 2}
//
//                     [genesis]
//                         |
//                        [0]
// idxToParent ==>      /     \
//                    [1]     [2]
//                    / \     / \
//                  [3] [4] [5] [6]
func NewBlockTestSet(t *testing.T, idxToParent []BlockID) (blocks map[BlockID]*core.Block) {
	blocks = make(map[BlockID]*core.Block)
	for i, parentID := range idxToParent {
		if parentID == GenesisID {
			blocks[GenesisID] = GenesisBlock
		}
		parentBlock, ok := blocks[parentID]
		require.True(t, ok)

		newBlock := NewTestBlock(t, parentBlock)
		newBlockID := BlockID(i)
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
	block, err := core.NewBlock(ChainID, coinbase, parent)
	require.Nil(t, err)
	require.NotNil(t, block)
	require.EqualValues(t, block.ParentHash(), parent.Hash())
	return block
}

// NewTestBlock return new block for test
func NewTestBlock(t *testing.T, parent *core.Block) *core.Block {
	block := getBlock(t, parent, "")
	err := block.Seal()
	require.Nil(t, err)
	return block
}

// NewTestBlockWithTxs return new block containing transactions
func NewTestBlockWithTxs(t *testing.T, parent *core.Block) *core.Block {
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
			MockAddress(t, ks),
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

// NewTransaction return new transaction
func NewTransaction(t *testing.T, fromKey signature.PrivateKey, toKey signature.PrivateKey, nonce uint64) *core.Transaction {
	from, err := common.PublicKeyToAddress(fromKey.PublicKey())
	require.Nil(t, err)
	to, err := common.PublicKeyToAddress(toKey.PublicKey())
	require.Nil(t, err)
	tx, err := core.NewTransaction(ChainID, from, to, util.Uint128Zero(), nonce, "", nil)
	require.Nil(t, err)
	return tx
}

// NewSignedTransaction return new signed transaction
func NewSignedTransaction(t *testing.T, from signature.PrivateKey, to signature.PrivateKey, nonce uint64) *core.Transaction {
	tx := NewTransaction(t, from, to, nonce)
	SignTx(t, tx, from)
	return tx
}

// NewPrivateKey return new private key
func NewPrivateKey(t *testing.T) signature.PrivateKey {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	require.NoError(t, err)
	return privKey
}

func newAddress(t *testing.T) common.Address {
	key := NewPrivateKey(t)
	addr, err := common.PublicKeyToAddress(key.PublicKey())
	require.NoError(t, err)
	return addr
}

func newNonce(t *testing.T) uint64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	require.NoError(t, err)
	return n.Uint64()
}

// NewRandomTransaction return new random transaction
func NewRandomTransaction(t *testing.T) *core.Transaction {
	from := NewPrivateKey(t)
	to := NewPrivateKey(t)
	tx := NewTransaction(t, from, to, newNonce(t))
	return tx
}

// NewRandomSignedTransaction return new random signed transaction
func NewRandomSignedTransaction(t *testing.T) *core.Transaction {
	from := NewPrivateKey(t)
	to := NewPrivateKey(t)
	tx := NewTransaction(t, from, to, newNonce(t))
	SignTx(t, tx, from)
	return tx
}

// SignTx sign transaction
func SignTx(t *testing.T, tx *core.Transaction, key signature.PrivateKey) {
	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	assert.NoError(t, err)
	sig.InitSign(key)
	tx.SignThis(sig)
}

// NewKeySlice return key slice
func NewKeySlice(t *testing.T, n int) []signature.PrivateKey {
	var keys []signature.PrivateKey
	for i := 0; i < n; i++ {
		keys = append(keys, NewPrivateKey(t))
	}
	return keys
}

// GetStorage return storage
func GetStorage(t *testing.T) storage.Storage {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	return s
}

// MockAddress return random generated address
func MockAddress(t *testing.T, ks *keystore.KeyStore) common.Address {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	assert.NoError(t, err)
	acc, err := ks.SetKey(privKey)
	assert.NoError(t, err)
	return acc
}

// NewTestTransactionManagers return new test transaction managers
func NewTestTransactionManagers(t *testing.T, n int) (mgrs []*core.TransactionManager, closeFn func()) {
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
			ChainId: ChainID,
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

type mockMedlet struct {
	config  *medletpb.Config
	storage storage.Storage
	genesis *corepb.Genesis
}

func NewMockMedlet(t *testing.T) *mockMedlet {
	stor, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	genesis, err := core.LoadGenesisConf(defaultGenesisConfPath)
	require.NoError(t, err)
	return &mockMedlet{
		config: &medletpb.Config{
			Chain: &medletpb.ChainConfig{
				ChainId: chainID,
			},
		},
		storage: stor,
		genesis: genesis,
	}
}

func (m *mockMedlet) Config() *medletpb.Config {
	return m.config
}

func (m *mockMedlet) Storage() storage.Storage {
	return m.storage
}

func (m *mockMedlet) Genesis() *corepb.Genesis {
	return m.genesis
}