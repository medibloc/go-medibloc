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

	goNet "net"

	"time"

	"fmt"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AddrKeyPair contains address and private key.
type AddrKeyPair struct {
	Addr    common.Address
	PrivKey signature.PrivateKey
}

// NewAddrKeyPair creates a pair of address and private key.
func NewAddrKeyPair(t *testing.T) *AddrKeyPair {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	require.NoError(t, err)

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
			Balance: "1000000000000000000",
			Vesting: "0",
			Vote:    []string{},
		})

		dynasties = append(dynasties, keypair)
		distributed = append(distributed, keypair)
	}

	for i := 0; i < 10; i++ {
		keypair := NewAddrKeyPair(t)
		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: keypair.Addr.Hex(),
			Balance: "1000000000000000000",
			Vesting: "0",
			Vote:    []string{},
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

// GetStorage return storage
func GetStorage(t *testing.T) storage.Storage {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	return s
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

//KeyOf find the key at which a given value can be found in the trie batch
func KeyOf(t *testing.T, trie *trie.Trie, value []byte) []byte {
	iter, err := trie.Iterator(nil)
	require.NoError(t, err)

	exist, err := iter.Next()
	for exist {
		require.NoError(t, err)
		if byteutils.Equal(iter.Value(), value) {
			return iter.Key()
		}
		exist, err = iter.Next()
	}
	return nil
}

//TrieLen counts the number of trie members
func TrieLen(t *testing.T, trie *trie.Batch) int {
	iter, err := trie.Iterator(nil)
	require.NoError(t, err)

	var cnt = 0
	exist, err := iter.Next()
	for exist {
		require.NoError(t, err)
		cnt++
		exist, err = iter.Next()
	}
	return cnt
}
