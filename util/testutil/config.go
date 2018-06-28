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
	"testing"

	"io/ioutil"

	"os"

	"net"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/stretchr/testify/require"
)

// NodeConfig is configuration for test node.
type NodeConfig struct {
	t   *testing.T
	dir string

	Config *medletpb.Config
	Miner  *AddrKeyPair

	Genesis   *corepb.Genesis
	Dynasties AddrKeyPairs
	TokenDist AddrKeyPairs
}

func defaultConfig(t *testing.T) *NodeConfig {
	cfg := &NodeConfig{
		t:      t,
		dir:    tempDir(t),
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
	return cfg.Config.Network.Listen
}

// SetPorts sets ports.
func (cfg *NodeConfig) SetPorts(baseport int) *NodeConfig {
	cfg.Config.Network.Listen = []string{fmt.Sprintf("127.0.0.1:%v", baseport)}
	cfg.Config.Rpc.RpcListen = []string{fmt.Sprintf("127.0.0.1:%v", baseport+1)}
	cfg.Config.Rpc.HttpListen = []string{fmt.Sprintf("127.0.0.1:%v", baseport+2)}
	return cfg
}

// SetRandomPorts sets random ports.
func (cfg *NodeConfig) SetRandomPorts() *NodeConfig {
	ports := FindRandomListenPorts(3)
	cfg.Config.Network.Listen = []string{fmt.Sprintf("127.0.0.1:%v", ports[0])}
	cfg.Config.Rpc.RpcListen = []string{fmt.Sprintf("127.0.0.1:%v", ports[1])}
	cfg.Config.Rpc.HttpListen = []string{fmt.Sprintf("127.0.0.1:%v", ports[2])}
	return cfg
}

// SetMiner sets miner.
func (cfg *NodeConfig) SetMiner(miner *AddrKeyPair) *NodeConfig {
	cfg.Miner = miner
	cfg.Config.Chain.Miner = miner.Address()
	cfg.Config.Chain.Passphrase = "passphrase"
	cfg.Config.Chain.StartMine = true
	cfg.Config.Chain.Coinbase = miner.Address()
	cfg.Config.Chain.Privkey = miner.PrivateKey()
	return cfg
}

// SetMinerFromDynasties chooses miner from dynasties.
func (cfg *NodeConfig) SetMinerFromDynasties(exclude []*AddrKeyPair) *NodeConfig {
	excludeMap := make(map[string]bool)
	for _, e := range exclude {
		excludeMap[e.Address()] = true
	}

	for _, d := range cfg.Dynasties {
		if _, ok := excludeMap[d.Address()]; !ok {
			cfg.SetMiner(d)
			return cfg
		}
	}

	require.True(cfg.t, false, "No miner left in dynasties.")
	return cfg
}

// SetRandomMiner sets random miner.
func (cfg *NodeConfig) SetRandomMiner() *NodeConfig {
	keypair := NewAddrKeyPair(cfg.t)
	return cfg.SetMiner(keypair)
}

// SetLog sets logging configuration.
func (cfg *NodeConfig) SetLog(path string, level string) *NodeConfig {
	cfg.Config.App.LogFile = path
	cfg.Config.App.LogLevel = level
	return cfg
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
	cfg.setGenesis(c.config.Genesis)
	cfg.Dynasties = c.config.Dynasties
	cfg.TokenDist = c.config.TokenDist
	return cfg
}

// SetSeed sets a seed node address.
func (cfg *NodeConfig) SetSeed(seed *Node) *NodeConfig {
	addrs := seed.config.GetListenAddrs()
	var seeds []string
	for _, addr := range addrs {
		_, port, err := net.SplitHostPort(addr)
		require.NoError(cfg.t, err)

		id := seed.med.NetService().Node().ID()
		s := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port, id)
		seeds = append(seeds, s)
	}
	cfg.Config.Network.Seed = seeds
	return cfg
}

// CleanUp cleans up data directories and configuration files.
func (cfg *NodeConfig) CleanUp() {
	require.NoError(cfg.t, os.RemoveAll("testdata"))
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
* Miner(addr/key)   : %v
* LogFile           : %v 
* LogLevel          : %v
* Dynasties         :
%v
* TokenDistribution :
%v
`
	return fmt.Sprintf(format,
		cfg.Config.Global.ChainId,
		cfg.Config.Global.Datadir,
		cfg.Config.Network.Seed,
		cfg.Config.Network.Listen,
		cfg.Config.Rpc.RpcListen,
		cfg.Config.Rpc.HttpListen,
		cfg.Miner,
		cfg.Config.App.LogFile,
		cfg.Config.App.LogLevel,
		cfg.Dynasties,
		cfg.TokenDist)
}

func tempDir(t *testing.T) string {
	err := os.MkdirAll("testdata", 0755)
	require.NoError(t, err)
	dir, err := ioutil.TempDir("testdata", "node")
	require.NoError(t, err)
	return dir
}
