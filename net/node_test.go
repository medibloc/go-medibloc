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

package net_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/medibloc/go-medibloc/medlet"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	netpb "github.com/medibloc/go-medibloc/net/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/medibloc/go-medibloc/util/testutil"
)

func TestBroadCastStorm(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer cleanup(t)

	seed, seedMsgCh := NewTestNode(t, ctx)
	node, nodeMsgCh := NewTestNodeWithSeed(t, ctx, seed)

	waitForConnection(t, 5*time.Second, seed, 1)

	seed.BroadcastMessage("bcast", TestMessageData, net.MessagePriorityNormal)
	msg := <-waitForMessage(1*time.Second, nodeMsgCh)
	require.NotNil(t, msg)
	require.Equal(t, "bcast", msg.MessageType())

	node.BroadcastMessage("bcast", TestMessageData, net.MessagePriorityNormal)
	msg = <-waitForMessage(1*time.Second, seedMsgCh)
	require.Nil(t, msg)
}

func TestTrimConnections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer cleanup(t)

	const (
		LowWaterMark  = 5
		HighWaterMark = 10
		GracePeriod   = 1
	)

	cfg := newDefalutConfig(t)
	cfg.Network.ConnMgrLowWaterMark = LowWaterMark
	cfg.Network.ConnMgrHighWaterMark = HighWaterMark
	cfg.Network.ConnMgrGracePeriod = GracePeriod

	seed, _ := NewTestNodeWithConfig(t, ctx, cfg)

	for i := 0; i < HighWaterMark; i++ {
		node, _ := NewTestNode(t, ctx)
		node.Peerstore().AddAddrs(seed.ID(), seed.Addrs(), peerstore.PermanentAddrTTL)
		node.SendMessageToPeer(TestMessageType, TestMessageData, net.MessagePriorityNormal, seed.ID().Pretty())
		fmt.Println(len(seed.Network().Conns()))
	}
	time.Sleep((GracePeriod + 3) * time.Second)
	seed.ConnManager().TrimOpenConns(ctx)
	assert.Equal(t, LowWaterMark, len(seed.Network().Conns()))
}

func TestLargeMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer cleanup(t)

	seed, seedMsgCh := NewTestNode(t, ctx)

	node, _ := NewTestNode(t, ctx)
	node.Peerstore().AddAddrs(seed.ID(), seed.Addrs(), peerstore.PermanentAddrTTL)

	testMsg := make([]byte, net.MaxMessageSize)
	node.SendMessageToPeer(TestMessageType, testMsg, net.MessagePriorityNormal, seed.ID().Pretty())
	recvMsg := <-waitForMessage(1*time.Second, seedMsgCh)
	require.Nil(t, recvMsg)

	testMsg = make([]byte, net.MaxMessageSize-13) // header size : 13
	node.SendMessageToPeer(TestMessageType, testMsg, net.MessagePriorityNormal, seed.ID().Pretty())
	recvMsg = <-waitForMessage(1*time.Second, seedMsgCh)
	require.NotNil(t, recvMsg)
}

func waitForMessage(timeout time.Duration, nodeRecvMsgCh chan net.Message) chan net.Message {
	recv := make(chan net.Message)
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case <-timer.C:
			recv <- nil
			return
		case msg := <-nodeRecvMsgCh:
			recv <- msg
			return
		}
	}()
	return recv
}

func waitForConnection(t *testing.T, timeout time.Duration, node *net.Node, targeNum int) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			t.Error("Timeout to reach target connection number ", targeNum)
			return
		case <-ticker.C:
			if len(node.Network().Conns()) >= targeNum {
				return
			}
		}
	}
}

func newDefalutConfig(t *testing.T) *medletpb.Config {
	cfg := medlet.DefaultConfig()
	cfg.Global.Datadir = testutil.TempDir(t)
	return cfg
}

func newConfigWithSeed(t *testing.T, seed *net.Node) *medletpb.Config {
	cfg := newDefalutConfig(t)
	pi := peerstore.PeerInfo{
		ID:    seed.ID(),
		Addrs: seed.Addrs(),
	}
	seeds := make([]*netpb.PeerInfo, 0)
	seeds = append(seeds, net.PeerInfoToProto(pi))
	cfg.Network.Seeds = seeds
	return cfg
}

func NewTestNode(t *testing.T, ctx context.Context) (*net.Node, chan net.Message) {
	return NewTestNodeWithConfig(t, ctx, newDefalutConfig(t))
}

func NewTestNodeWithSeed(t *testing.T, ctx context.Context, seed *net.Node) (*net.Node, chan net.Message) {
	cfg := newConfigWithSeed(t, seed)
	return NewTestNodeWithConfig(t, ctx, cfg)
}

func NewTestNodeWithConfig(t *testing.T, ctx context.Context, cfg *medletpb.Config) (*net.Node, chan net.Message) {
	receiveMsgCh := make(chan net.Message)
	node, err := net.NewNode(ctx, cfg, receiveMsgCh)
	require.NoError(t, err)
	require.NoError(t, node.Start()) // start net service

	return node, receiveMsgCh
}

func cleanup(t *testing.T) {
	require.NoError(t, os.RemoveAll("testdata"))
}
