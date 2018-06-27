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

	"github.com/medibloc/go-medibloc/medlet"
	"github.com/stretchr/testify/require"
)

// Node is node for testing.
type Node struct {
	mu sync.RWMutex
	t  *testing.T

	med    *medlet.Medlet
	config *NodeConfig

	started bool
}

// NewNode creates node for test.
func NewNode(t *testing.T, cfg *NodeConfig) *Node {
	med, err := medlet.New(cfg.Config)
	require.Nil(t, err)
	return &Node{
		t:       t,
		med:     med,
		config:  cfg,
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

	err := node.med.Setup()
	require.NoError(node.t, err)

	err = node.med.Start()
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

	node.med.Stop()
}

// IsStarted returns whether it has been started.
func (node *Node) IsStarted() bool {
	node.mu.RLock()
	defer node.mu.RUnlock()

	return node.started
}

// String returns summary of test node.
func (node *Node) String() string {
	return node.config.String()
}

// Network is set of nodes.
type Network struct {
	t *testing.T

	Seed  *Node
	Nodes []*Node
}

// NewNetwork creates network.
func NewNetwork(t *testing.T) *Network {
	return &Network{
		t: t,
	}
}

// NewSeedNode creates seed node.
func (n *Network) NewSeedNode() *Node {
	cfg := NewConfig(n.t).
		SetRandomGenesis()

	node := NewNode(n.t, cfg)
	n.Seed = node
	n.Nodes = append(n.Nodes, node)
	return node
}

// NewNode creates node.
func (n *Network) NewNode() *Node {
	require.NotNil(n.t, n.Seed)
	require.True(n.t, len(n.Nodes) > 0)

	cfg := NewConfig(n.t).
		SetGenesisFrom(n.Seed).
		SetSeed(n.Seed)

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
		for int(node.med.NetService().Node().EstablishedPeersCount()) != len(n.Nodes)-1 {
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
		node.config.CleanUp()
	}
}
