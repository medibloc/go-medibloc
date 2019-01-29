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
	"context"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"sync"
	"time"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// Node is node for testing.
type Node struct {
	mu sync.RWMutex
	t  *testing.T

	ctx    context.Context
	cancel context.CancelFunc
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

	ctx, cancel := context.WithCancel(context.Background())
	node.cancel = cancel

	err = node.Med.Start(ctx)
	require.NoError(node.t, err)

	startTime := time.Now()
	for {
		require.True(node.t, time.Now().Sub(startTime) < time.Duration(3*time.Second))
		conn, err := net.Dial("tcp", node.Config.Config.Rpc.HttpListen[0])
		if err != nil {
			time.Sleep(10 * time.Millisecond)
		} else {
			require.NotNil(node.t, conn)
			conn.Close()
			return
		}
	}
}

// Stop stops test node.
func (node *Node) Stop() {
	node.mu.Lock()
	defer node.mu.Unlock()

	if !node.started {
		return
	}
	node.started = false

	if node.cancel != nil {
		node.cancel()
	}
	node.Med.Stop()
}

// Restart restart node
func (node *Node) Restart() {
	node.Stop()

	med, err := medlet.New(node.Config.Config)
	require.Nil(node.t, err)

	node.Med = med
	node.Start()
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

// WaitUntilTailHeight waits until blockchain has a designated height or timeout
func (node *Node) WaitUntilTailHeight(height uint64, timeLimit time.Duration) error {
	timeout := time.NewTimer(timeLimit)
	defer timeout.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout.C:
			return ErrExecutionTimeout
		case <-ticker.C:
			if node.Med.BlockManager().TailBlock().Height() == height {
				return nil
			}
		}
	}
}

// WaitUntilBlockAcceptedOnChain waits until the block is accepted on the blockchain
func (node *Node) WaitUntilBlockAcceptedOnChain(hash []byte, timeLimit time.Duration) error {
	timeout := time.NewTimer(timeLimit)
	defer timeout.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout.C:
			return ErrExecutionTimeout
		case <-ticker.C:
			if node.Med.BlockManager().BlockByHash(hash) != nil {
				return nil
			}
		}
	}
}

// Network is set of nodes.
type Network struct {
	t       *testing.T
	logHook *test.Hook

	DynastySize int
	Seed        *Node
	Nodes       []*Node
}

// NewNetworkWithDynastySize creates network.
func NewNetwork(t *testing.T) *Network {
	return NewNetworkWithDynastySize(t, blockutil.DynastySize)
}

// NewNetworkWithDynastySize creates network with dynasty size.
func NewNetworkWithDynastySize(t *testing.T, dynastySize int) *Network {
	logHook := logging.InitTestLogger(filepath.Join("testdata", t.Name()))
	return &Network{
		t:           t,
		logHook:     logHook,
		DynastySize: dynastySize,
	}
}

// NewSeedNode creates seed node.
func (n *Network) NewSeedNode() *Node {
	cfg := NewConfig(n.t).
		SetRandomGenesis(n.DynastySize)
	return n.NewSeedNodeWithConfig(cfg)
}

// NewSeedNodeWithConfig creates seed node.
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

// NewNodeWithConfig creates node with custom config
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
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for _, node := range n.Nodes {
		node.Med.NetService().Node().DHTSync()
		for len(node.Med.NetService().Node().Peerstore().Peers()) != len(n.Nodes) {
			select {
			case <-timer.C:
				n.t.Error("Failed to wait for established")
			case <-ticker.C:
				node.Med.NetService().Node().DHTSync()
			}
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
	n.logHook.Reset()

	size, err := DirSize(filepath.Join("testdata", n.t.Name()))
	require.NoError(n.t, err)

	if n.t.Failed() {
		n.t.Log("TestData size:", size)
		wd, err := os.Getwd()
		require.NoError(n.t, err)
		n.t.Logf("Test Failed. LogDir:%s/%s", wd, filepath.Join("testdata", n.t.Name()))
		logdata, err := ioutil.ReadFile(filepath.Join("testdata", n.t.Name(), "medibloc.log"))
		require.NoError(n.t, err)
		n.t.Log(string(logdata))
		return
	}

	require.NoError(n.t, os.RemoveAll(filepath.Join("testdata", n.t.Name())))

	infos, err := ioutil.ReadDir("testdata")
	require.NoError(n.t, err)
	if len(infos) == 0 {
		require.NoError(n.t, os.RemoveAll("testdata"))
	}
}

// LogTestHook returns test hook for log messages.
func (n *Network) LogTestHook() *test.Hook {
	return n.logHook
}

// SetProposerFromDynasties chooses proposer from dynasties.
func (n *Network) SetProposerFromDynasties(node *Node) {
	require.False(n.t, node.IsStarted())

	exclude := n.assignedProposers()
	node.Config.SetProposerFromDynasties(exclude)
}

// SetRandomProposer sets random proposer.
func (n *Network) SetRandomProposer(node *Node) {
	require.False(n.t, node.IsStarted())

	node.Config.SetRandomProposer()
}

func (n *Network) assignedProposers() []*keyutil.AddrKeyPair {
	proposers := make([]*keyutil.AddrKeyPair, 0)
	for _, node := range n.Nodes {
		if node.Config.Proposer != nil {
			proposers = append(proposers, node.Config.Proposer)
		}
	}
	return proposers
}

// FindProposer returns block proposer for time stamp
func (n *Network) FindProposer(ts int64, parent *core.Block) *keyutil.AddrKeyPair {
	dynasties := n.Seed.Config.Dynasties
	d := n.Seed.Med.Consensus()
	proposer, err := d.FindMintProposer(ts, parent)
	require.Nil(n.t, err)
	v := dynasties.FindPair(proposer)
	require.NotNil(n.t, v, "Failed to find proposer's privateKey")
	return v
}
