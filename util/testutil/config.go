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
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package testutil

import (
	"fmt"
	"path/filepath"
	"testing"

	multiaddr "github.com/multiformats/go-multiaddr"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/medibloc/go-medibloc/net"

	"io/ioutil"

	"os"

	"github.com/gogo/protobuf/proto"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/stretchr/testify/require"
)

// NodeConfig is configuration for test node.
type NodeConfig struct {
	t   *testing.T
	dir string

	Config   *medletpb.Config
	Proposer *AddrKeyPair

	Genesis   *corepb.Genesis
	Dynasties AddrKeyPairs
	TokenDist AddrKeyPairs
}

func defaultConfig(t *testing.T) *NodeConfig {
	cfg := &NodeConfig{
		t:      t,
		dir:    TempDir(t),
		Config: medlet.DefaultConfig(),
	}
	return cfg.SetRandomPorts().SetRandomDataDir()
}

// NewConfig returns configuration of node.
func NewConfig(t *testing.T) *NodeConfig {
	return defaultConfig(t)
}

// SetChainID sets chain ID.
func (cfg *NodeConfig) SetChainID(chainID uint32) *NodeConfig {
	cfg.Config.Global.ChainId = chainID
	return cfg
}

// SetDataDir sets data dir.
func (cfg *NodeConfig) SetDataDir(dir string) *NodeConfig {
	cfg.Config.Global.Datadir = dir
	return cfg
}

// SetRandomDataDir sets random data dir.
func (cfg *NodeConfig) SetRandomDataDir() *NodeConfig {
	dir := fmt.Sprintf("%s/data", cfg.dir)
	err := os.MkdirAll(dir, 0755)
	require.NoError(cfg.t, err)
	return cfg.SetDataDir(dir)
}

// GetListenAddrs returns listen address of a node.
func (cfg *NodeConfig) GetListenAddrs() []string {
	return cfg.Config.Network.Listens
}

// SetPorts sets ports.
func (cfg *NodeConfig) SetPorts(baseport int) *NodeConfig {
	cfg.Config.Network.Listens = []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%v", baseport)}
	cfg.Config.Rpc.RpcListen = []string{fmt.Sprintf("127.0.0.1:%v", baseport+1)}
	cfg.Config.Rpc.HttpListen = []string{fmt.Sprintf("127.0.0.1:%v", baseport+2)}
	return cfg
}

// SetRandomPorts sets random ports.
func (cfg *NodeConfig) SetRandomPorts() *NodeConfig {
	ports := FindRandomListenPorts(3)
	cfg.Config.Network.Listens = []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/%v", ports[0])}
	cfg.Config.Rpc.RpcListen = []string{fmt.Sprintf("127.0.0.1:%v", ports[1])}
	cfg.Config.Rpc.HttpListen = []string{fmt.Sprintf("127.0.0.1:%v", ports[2])}
	return cfg
}

// SetProposer sets proposer.
func (cfg *NodeConfig) SetProposer(proposer *AddrKeyPair) *NodeConfig {
	cfg.Proposer = proposer
	cfg.Config.Chain.StartMine = true
	cfg.Config.Chain.Privkey = proposer.PrivateKey()

	pCfg := &medletpb.ProposerConfig{
		Proposer: proposer.Address(),
		Privkey:  proposer.PrivateKey(),
		Coinbase: proposer.Address(),
	}
	cfg.Config.Chain.Proposers = append(cfg.Config.Chain.Proposers, pCfg)
	return cfg
}

// SetProposerFromDynasties chooses proposer from dynasties.
func (cfg *NodeConfig) SetProposerFromDynasties(exclude []*AddrKeyPair) *NodeConfig {
	excludeMap := make(map[string]bool)
	for _, e := range exclude {
		excludeMap[e.Address()] = true
	}

	for _, d := range cfg.Dynasties {
		if _, ok := excludeMap[d.Address()]; !ok {
			cfg.SetProposer(d)
			return cfg
		}
	}

	require.True(cfg.t, false, "No proposer left in dynasties.")
	return cfg
}

// SetRandomProposer sets random proposer.
func (cfg *NodeConfig) SetRandomProposer() *NodeConfig {
	keypair := NewAddrKeyPair(cfg.t)
	return cfg.SetProposer(keypair)
}

// setGenesis sets genesis configuration.
func (cfg *NodeConfig) setGenesis(genesis *corepb.Genesis) *NodeConfig {
	cfg.Genesis = genesis
	path := cfg.writeGenesisConfig()
	cfg.Config.Chain.Genesis = path
	return cfg
}

// SetRandomGenesis sets random genesis configuration.
func (cfg *NodeConfig) SetRandomGenesis(dynastySize int) *NodeConfig {
	genesis, dynasties, tokenDist := NewTestGenesisConf(cfg.t, dynastySize)
	cfg.setGenesis(genesis)
	cfg.Dynasties = dynasties
	cfg.TokenDist = tokenDist
	return cfg
}

// SetGenesisFrom sets genesis configruation from other node's config.
func (cfg *NodeConfig) SetGenesisFrom(c *Node) *NodeConfig {
	cfg.setGenesis(c.Config.Genesis)
	cfg.Dynasties = c.Config.Dynasties
	cfg.TokenDist = c.Config.TokenDist
	return cfg
}

// SetSeed sets a seed node address.
func (cfg *NodeConfig) SetSeed(seed *Node) *NodeConfig {

	addrs := make([]multiaddr.Multiaddr, 0)
	for _, v := range seed.Med.NetService().Node().Network().ListenAddresses() {
		if v.String() == "/p2p-circuit" {
			continue
		}
		addrs = append(addrs, v)
	}

	seedInfo := peerstore.PeerInfo{
		ID:    seed.Med.NetService().Node().ID(),
		Addrs: addrs,
	}
	cfg.Config.Network.Seeds = net.PeersToProto(seedInfo)
	return cfg
}

//DirSize returns dir size
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func dumpGenesisConfig(genesis *corepb.Genesis) string {
	return proto.MarshalTextString(genesis)
}

func (cfg *NodeConfig) writeGenesisConfig() (path string) {
	path = fmt.Sprintf("%s/genesis.conf", cfg.dir)
	f, err := os.Create(path)
	require.NoError(cfg.t, err)
	defer f.Close()

	_, err = f.WriteString(dumpGenesisConfig(cfg.Genesis))
	require.NoError(cfg.t, err)

	return path
}

// String returns summary of test config.
func (cfg *NodeConfig) String() string {
	format := `
* ChainID           : %v
* DataDir           : %v
* Seed              : %v
* Net               : %v
* RPC               : %v
* HTTP              : %v
* Proposer(addr/key)   : %v
* Dynasties         :
%v
* TokenDistribution :
%v
`
	return fmt.Sprintf(format,
		cfg.Config.Global.ChainId,
		cfg.Config.Global.Datadir,
		cfg.Config.Network.Seeds,
		cfg.Config.Network.Listens,
		cfg.Config.Rpc.RpcListen,
		cfg.Config.Rpc.HttpListen,
		cfg.Proposer,
		cfg.Dynasties,
		cfg.TokenDist)
}

//TempDir make temporary directory for test
func TempDir(t *testing.T) string {
	err := os.MkdirAll(filepath.Join("testdata", t.Name()), 0755)
	require.NoError(t, err)
	dir, err := ioutil.TempDir(filepath.Join("testdata", t.Name()), "node")
	require.NoError(t, err)
	return dir
}
