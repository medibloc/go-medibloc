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
	"testing"

	"sync"
	"time"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// Node is node for testing.
type Node struct {
	mu sync.RWMutex
	t  *testing.T

	Med    *medlet.Medlet
	Config *NodeConfig

	started bool
}

// NewNode creates node for test.
func NewNode(t *testing.T, cfg *NodeConfig) *Node {
	med, err := medlet.New(cfg.Config)
	require.Nil(t, err)
	return &Node{
		t:       t,
		Med:     med,
		Config:  cfg,
		started: false,
	}
}

// Start starts test node.
func (node *Node) Start() {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.started {
		return
	}
	node.started = true

	err := node.Med.Setup()
	require.NoError(node.t, err)

	err = node.Med.Start()
	require.NoError(node.t, err)

}

// Stop stops test node.
func (node *Node) Stop() {
	node.mu.Lock()
	defer node.mu.Unlock()

	if !node.started {
		return
	}
	node.started = false

	node.Med.Stop()
}

// IsStarted returns whether it has been started.
func (node *Node) IsStarted() bool {
	node.mu.RLock()
	defer node.mu.RUnlock()

	return node.started
}

// String returns summary of test node.
func (node *Node) String() string {
	return node.Config.String()
}

// GenesisBlock returns genesis block.
func (node *Node) GenesisBlock() *core.Block {
	block, err := node.Med.BlockManager().BlockByHeight(core.GenesisHeight)
	require.NoError(node.t, err)
	return block
}

// Tail returns tail block.
func (node *Node) Tail() *core.Block {
	block := node.Med.BlockManager().TailBlock()
	return block
}

// Network is set of nodes.
type Network struct {
	t *testing.T

	DynastySize int
	Seed        *Node
	Nodes       []*Node
}

// NewNetwork creates network.
func NewNetwork(t *testing.T, dynastySize int) *Network {
	logging.Init("testdata", "debug", 0)
	return &Network{
		t:           t,
		DynastySize: dynastySize,
	}
}

// NewSeedNode creates seed node.
func (n *Network) NewSeedNode() *Node {
	cfg := NewConfig(n.t).
		SetRandomGenesis(n.DynastySize)
	return n.NewSeedNodeWithConfig(cfg)
}

// NewSeedNode creates seed node.
func (n *Network) NewSeedNodeWithConfig(cfg *NodeConfig) *Node {
	node := NewNode(n.t, cfg)
	n.Seed = node
	n.Nodes = append(n.Nodes, node)
	return node
}

// NewNode creates node.
func (n *Network) NewNode() *Node {
	return n.NewNodeWithConfig(NewConfig(n.t))
}

func (n *Network) NewNodeWithConfig(cfg *NodeConfig) *Node {
	require.NotNil(n.t, n.Seed)
	require.True(n.t, len(n.Nodes) > 0)

	cfg.SetGenesisFrom(n.Seed).SetSeed(n.Seed)

	node := NewNode(n.t, cfg)
	n.Nodes = append(n.Nodes, node)
	return node

}

// Start starts nodes in network.
func (n *Network) Start() {
	if !n.Seed.IsStarted() {
		n.Seed.Start()
	}
	for _, node := range n.Nodes {
		if !node.IsStarted() {
			node.Start()
		}
	}
}

// WaitForEstablished waits until connections between peers are established.
func (n *Network) WaitForEstablished() {
	for _, node := range n.Nodes {
		for int(node.Med.NetService().Node().EstablishedPeersCount()) != len(n.Nodes)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// Stop stops nodes in network
func (n *Network) Stop() {
	for _, node := range n.Nodes {
		node.Stop()
	}
}

// Cleanup cleans up directories and files.
func (n *Network) Cleanup() {
	n.Stop()
	for _, node := range n.Nodes {
		node.Config.CleanUp()
	}
}

// SetLogTestHook sets test hook for log messages.
func (n *Network) SetLogTestHook() *test.Hook {
	logging.SetNullLogger()
	return logging.SetTestHook()
}

// SetMinerFromDynasties chooses miner from dynasties.
func (n *Network) SetMinerFromDynasties(node *Node) {
	require.False(n.t, node.IsStarted())

	exclude := n.assignedMiners()
	node.Config.SetMinerFromDynasties(exclude)
}

// SetRandomMiner sets random miner.
func (n *Network) SetRandomMiner(node *Node) {
	require.False(n.t, node.IsStarted())

	node.Config.SetRandomMiner()
}

func (n *Network) assignedMiners() []*AddrKeyPair {
	miners := make([]*AddrKeyPair, 0)
	for _, node := range n.Nodes {
		if node.Config.Miner != nil {
			miners = append(miners, node.Config.Miner)
		}
	}
	return miners
}

func (n *Network) FindProposer(ts int64, parent *core.Block) *AddrKeyPair {
	dynasties := n.Seed.Config.Dynasties
	d := n.Seed.Med.Consensus()
	proposer, err := d.FindMintProposer(ts, parent)
	require.Nil(n.t, err)
	v := dynasties.FindPair(proposer)
	require.NotNil(n.t, v, "Failed to find proposer's privateKey")
	return v
}
