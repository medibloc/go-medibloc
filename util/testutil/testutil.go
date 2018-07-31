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

package testutil

import (
	"strings"
	"testing"

	"crypto/rand"
	"math/big"

	goNet "net"

	"time"

	"fmt"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BlockID block ID
type BlockID int

var (
	// GenesisID genesis ID
	GenesisID BlockID = -1
)

// AddrKeyPair contains address and private key.
type AddrKeyPair struct {
	Addr    common.Address
	PrivKey signature.PrivateKey
}

// NewAddrKeyPair creates a pair of address and private key.
func NewAddrKeyPair(t *testing.T) *AddrKeyPair {
	privKey := NewPrivateKey(t)
	addr, err := common.PublicKeyToAddress(privKey.PublicKey())
	require.NoError(t, err)
	return &AddrKeyPair{
		Addr:    addr,
		PrivKey: privKey,
	}
}

// Address returns address.
func (pair *AddrKeyPair) Address() string {
	return pair.Addr.Hex()
}

// PrivateKey returns private key.
func (pair *AddrKeyPair) PrivateKey() string {
	d, _ := pair.PrivKey.Encoded()
	return byteutils.Bytes2Hex(d)
}

// String describes AddrKeyPair in string format.
func (pair *AddrKeyPair) String() string {
	if pair == nil {
		return ""
	}
	return fmt.Sprintf("Addr:%v, PrivKey:%v\n", pair.Address(), pair.PrivateKey())
}

// AddrKeyPairs is a slice of AddrKeyPair structure.
type AddrKeyPairs []*AddrKeyPair

// FindPrivKey finds private key of given address.
func (pairs AddrKeyPairs) FindPrivKey(addr common.Address) signature.PrivateKey {
	for _, dynasty := range pairs {
		if dynasty.Addr.Equals(addr) {
			return dynasty.PrivKey
		}
	}
	return nil
}

// FindPair finds AddrKeyPair of given address.
func (pairs AddrKeyPairs) FindPair(addr common.Address) *AddrKeyPair {
	for _, dynasty := range pairs {
		if dynasty.Addr.Equals(addr) {
			return dynasty
		}
	}
	return nil
}

// NewTestGenesisConf returns a genesis configuration for tests.
func NewTestGenesisConf(t *testing.T, dynastySize int) (conf *corepb.Genesis, dynasties AddrKeyPairs, distributed AddrKeyPairs) {
	conf = &corepb.Genesis{
		Meta: &corepb.GenesisMeta{
			ChainId:     ChainID,
			DynastySize: uint32(dynastySize),
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

	for i := 0; i < dynastySize; i++ {
		keypair := NewAddrKeyPair(t)
		dynasty = append(dynasty, keypair.Addr.Hex())
		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: keypair.Addr.Hex(),
			Value:   "1000000000",
		})

		dynasties = append(dynasties, keypair)
		distributed = append(distributed, keypair)
	}

	for i := 0; i < 10; i++ {
		keypair := NewAddrKeyPair(t)
		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: keypair.Addr.Hex(),
			Value:   "1000000000",
		})
		distributed = append(distributed, keypair)
	}

	conf.Consensus.Dpos.Dynasty = dynasty
	conf.TokenDistribution = tokenDist
	return conf, dynasties, distributed
}

// NewTestGenesisBlock returns a genesis block for tests.
func NewTestGenesisBlock(t *testing.T, dynastySize int) (genesis *core.Block, dynasties AddrKeyPairs, distributed AddrKeyPairs) {
	conf, dynasties, distributed := NewTestGenesisConf(t, dynastySize)
	s, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	d := dpos.New(dynastySize)
	genesis, err = core.NewGenesisBlock(conf, d, s)
	require.NoError(t, err)

	return genesis, dynasties, distributed
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
	block.SetTimestamp(parentBlockTime.Add(dpos.BlockInterval).Unix())
	require.NoError(t, err)

	return block
}

// NewTestBlock return new block for test
func NewTestBlock(t *testing.T, parent *core.Block) *core.Block {
	block := getBlock(t, parent, "")
	//require.Nil(t, block.State().TransitionDynasty(block.Timestamp()))
	err := block.Seal()
	require.Nil(t, err)
	return block
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

// MockMedlet sets up components for tests.
type MockMedlet struct {
	config             *medletpb.Config
	genesis            *corepb.Genesis
	netService         net.Service
	storage            storage.Storage
	blockManager       *core.BlockManager
	transactionManager *core.TransactionManager
	consensus          core.Consensus
	dynasties          AddrKeyPairs
}

// NewMockMedlet returns MockMedlet.
func NewMockMedlet(t *testing.T) *MockMedlet {
	cfg := medlet.DefaultConfig()
	cfg.Chain.BlockCacheSize = 1
	cfg.Chain.Coinbase = "02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c"
	cfg.Chain.Miner = "02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c"

	genesisConf, dynasties, _ := NewTestGenesisConf(t, 21)
	var ns net.Service
	stor, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	consensus := dpos.New(int(genesisConf.Meta.DynastySize))
	bm, err := core.NewBlockManager(cfg)
	require.NoError(t, err)
	bm.InjectEmitter(core.NewEventEmitter(128))
	tm := core.NewTransactionManager(cfg)
	require.NoError(t, err)
	tm.InjectEmitter(core.NewEventEmitter(128))

	err = bm.Setup(genesisConf, stor, ns, consensus)
	require.NoError(t, err)
	tm.Setup(ns)
	err = consensus.Setup(cfg, genesisConf, bm, tm)
	require.NoError(t, err)

	return &MockMedlet{
		config:             cfg,
		genesis:            genesisConf,
		netService:         ns,
		storage:            stor,
		blockManager:       bm,
		transactionManager: tm,
		consensus:          consensus,
		dynasties:          dynasties,
	}
}

// Config returns config.
func (m *MockMedlet) Config() *medletpb.Config {
	return m.config
}

// Genesis return genesis configuration.
func (m *MockMedlet) Genesis() *corepb.Genesis {
	return m.genesis
}

// NetService returns net.Service.
func (m *MockMedlet) NetService() net.Service {
	return m.netService
}

// Storage return storage.
func (m *MockMedlet) Storage() storage.Storage {
	return m.storage
}

// BlockManager returns BlockManager.
func (m *MockMedlet) BlockManager() *core.BlockManager {
	return m.blockManager
}

// TransactionManager returns TransactionManager.
func (m *MockMedlet) TransactionManager() *core.TransactionManager {
	return m.transactionManager
}

// Consensus returns Consensus.
func (m *MockMedlet) Consensus() core.Consensus {
	return m.consensus
}

// Dynasties returns Dynasties.
func (m *MockMedlet) Dynasties() AddrKeyPairs {
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
		for {
			conn, err := goNet.DialTimeout("tcp", listens[i].Addr().String(), time.Millisecond*50)
			if err != nil {
				break
			}
			conn.Close()
			time.Sleep(time.Millisecond * 50)
		}
	}

	return ports
}
