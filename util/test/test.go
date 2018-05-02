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
	"testing"

	"crypto/rand"
	"math/big"

	"time"

	goNet "net"
	"strings"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/keystore"
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
	// GenesisID genesis ID
	GenesisID BlockID = -1

	fromAddress = "02279dcbc360174b4348685e75287a60abc5290497d2e3330b6a1791c4f35bcd20"
)

// AddrKeyPair contains address and private key.
type AddrKeyPair struct {
	Addr    common.Address
	PrivKey signature.PrivateKey
}

// Dynasties is a slice of dynasties.
type Dynasties []*AddrKeyPair

func (d Dynasties) findPrivKey(addr common.Address) signature.PrivateKey {
	for _, dynasty := range d {
		if dynasty.Addr.Equals(addr) {
			return dynasty.PrivKey
		}
	}
	return nil
}

// NewTestGenesisConf returns a genesis configuration for tests.
func NewTestGenesisConf(t *testing.T) (conf *corepb.Genesis, dynasties Dynasties) {
	conf = &corepb.Genesis{
		Meta: &corepb.GenesisMeta{
			ChainId: ChainID,
		},
		Consensus: &corepb.GenesisConsensus{
			Dpos: &corepb.GenesisConsensusDpos{
				Dynasty: nil,
			},
		},
		TokenDistribution: nil,
	}

	var dynasty []string
	var tokenDist []*corepb.GenesisTokenDistribution
	for i := 0; i < dpos.DynastySize; i++ {
		privKey := NewPrivateKey(t)
		addr, err := common.PublicKeyToAddress(privKey.PublicKey())
		require.NoError(t, err)
		dynasty = append(dynasty, addr.Hex())
		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: addr.Hex(),
			Value:   "1000000000",
		})

		dynasties = append(dynasties, &AddrKeyPair{
			Addr:    addr,
			PrivKey: privKey,
		})
	}

	conf.Consensus.Dpos.Dynasty = dynasty
	conf.TokenDistribution = tokenDist
	return conf, dynasties
}

// NewTestGenesisBlock returns a genesis block for tests.
func NewTestGenesisBlock(t *testing.T) (genesis *core.Block, dynasties Dynasties) {
	conf, dynasties := NewTestGenesisConf(t)
	s, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	genesis, err = core.NewGenesisBlock(conf, s)
	require.NoError(t, err)

	return genesis, dynasties
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
func NewBlockTestSet(t *testing.T, genesis *core.Block, idxToParent []BlockID) (blocks map[BlockID]*core.Block) {
	blocks = make(map[BlockID]*core.Block)
	for i, parentID := range idxToParent {
		if parentID == GenesisID {
			blocks[GenesisID] = genesis
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

	parentBlockTime := time.Unix(parent.Timestamp(), 0)
	err = block.SetTimestamp(parentBlockTime.Add(dpos.BlockInterval).Unix())
	require.NoError(t, err)

	return block
}

// SignBlock signs block.
func SignBlock(t *testing.T, block *core.Block, dynasties Dynasties) {
	members, err := block.State().Dynasty()
	require.NoError(t, err)
	proposer, err := dpos.FindProposer(block.Timestamp(), members)
	require.NoError(t, err)

	privKey := dynasties.findPrivKey(*proposer)
	require.NotNil(t, privKey)

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	require.NoError(t, err)
	sig.InitSign(privKey)
	block.SignThis(sig)
}

// NewTestBlock return new block for test
func NewTestBlock(t *testing.T, parent *core.Block) *core.Block {
	block := getBlock(t, parent, "")
	err := block.Seal()
	require.Nil(t, err)
	return block
}

// NewTestBlockWithTxs return new block containing transactions
func NewTestBlockWithTxs(t *testing.T, parent *core.Block, from *AddrKeyPair) *core.Block {
	block := getBlock(t, parent, fromAddress)
	txs := newTestTransactions(t, block, from)
	block.SetTransactions(txs)
	return block
}

func newTestTransactions(t *testing.T, block *core.Block, from *AddrKeyPair) core.Transactions {
	ks := keystore.NewKeyStore()
	txDatas := []struct {
		from    common.Address
		to      common.Address
		amount  *util.Uint128
		privKey signature.PrivateKey
	}{
		{
			from.Addr,
			MockAddress(t, ks),
			util.NewUint128FromUint(10),
			from.PrivKey,
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
		assert.NoError(t, err)
		sig.InitSign(txData.privKey)
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

	cfg := &medletpb.Config{
		Global: &medletpb.GlobalConfig{
			ChainId: ChainID,
		},
	}
	for i := 0; i < n; i++ {
		mgr := core.NewTransactionManager(cfg)
		mgr.Setup(svc[i])
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

// MockMedlet sets up components for tests.
type MockMedlet struct {
	config    *medletpb.Config
	storage   storage.Storage
	genesis   *corepb.Genesis
	ns        net.Service
	bm        *core.BlockManager
	tm        *core.TransactionManager
	consensus core.Consensus
	dynasties Dynasties
}

// NewMockMedlet returns MockMedlet.
func NewMockMedlet(t *testing.T) *MockMedlet {
	cfg := &medletpb.Config{
		Global:&medletpb.GlobalConfig{
			ChainId:  ChainID,
		},
		Chain: &medletpb.ChainConfig{
			Coinbase: "02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c",
			Miner:    "02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c",
		},
	}

	var ns net.Service
	stor, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	genesisConf, dynasties := NewTestGenesisConf(t)
	require.NoError(t, err)
	consensus := dpos.New(cfg)
	bm, err := core.NewBlockManager(cfg)
	require.NoError(t, err)
	tm := core.NewTransactionManager(cfg)
	require.NoError(t, err)

	err = bm.Setup(genesisConf, stor, ns, consensus)
	require.NoError(t, err)
	tm.Setup(ns)
	consensus.Setup(bm, tm)

	return &MockMedlet{
		config:    cfg,
		storage:   stor,
		genesis:   genesisConf,
		ns:        ns,
		bm:        bm,
		tm:        tm,
		consensus: consensus,
		dynasties: dynasties,
	}
}

// Config returns config.
func (m *MockMedlet) Config() *medletpb.Config {
	return m.config
}

// Storage return storage.
func (m *MockMedlet) Storage() storage.Storage {
	return m.storage
}

// Genesis return genesis configuration.
func (m *MockMedlet) Genesis() *corepb.Genesis {
	return m.genesis
}

// NetService returns net.Service.
func (m *MockMedlet) NetService() net.Service {
	return m.ns
}


// Consensus returns Consensus.
func (m *MockMedlet) Consensus() core.Consensus {
	return m.consensus
}

// TransactionManager returns TransactionManager.
func (m *MockMedlet) TransactionManager() *core.TransactionManager {
	return m.tm
}

// BlockManager returns BlockManager.
func (m *MockMedlet) BlockManager() *core.BlockManager {
	return m.bm
}

// Dynasties returns Dynasties.
func (m *MockMedlet) Dynasties() Dynasties {
	return m.dynasties
}

//FindRandomListenPorts returns empty ports
func FindRandomListenPorts(n int) (ports []string) {
	listens := make([]goNet.Listener, 0)
	for i := 0; i < n; i++ {
		lis, _ := goNet.Listen("tcp", ":0")
		addr := lis.Addr().String()
		ports = append(ports, strings.TrimLeft(addr, "[::]"))
		listens = append(listens, lis)
	}
	for i := 0; i < n; i++ {
		listens[i].Close()
	}
	return ports
}