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

	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/stretchr/testify/require"

	"github.com/medibloc/go-medibloc/util/testutil"
)

func TestManyConnections(t *testing.T) {
	logging.SetTestHook()
	const NumberOfNet = 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer cleanup(t)

	seed, _ := NewTestNode(t, ctx)

	for i := 0; i < NumberOfNet; i++ {
		node, _ := NewTestNode(t, ctx)
		node.Peerstore().AddAddrs(seed.ID(), seed.Addrs(), peerstore.PermanentAddrTTL)
		node.SendMessageToPeer(TestMessageType, TestMessageData, net.MessagePriorityNormal, seed.ID().Pretty())
	}

	//time.Sleep(net.DefaultConnMgrGracePeriod)
	t.Log(seed.PeersCount())
}

func TestLargeMessage(t *testing.T) {
	//logging.SetTestHook()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer cleanup(t)

	seed, seedMsgCh := NewTestNode(t, ctx)

	node, _ := NewTestNode(t, ctx)
	node.Peerstore().AddAddrs(seed.ID(), seed.Addrs(), peerstore.PermanentAddrTTL)

	largeMessage := make([]byte, net.MaxMessageSize/2)

	node.SendMessageToPeer(TestMessageType, largeMessage, net.MessagePriorityNormal, seed.ID().Pretty())
	msg := <-seedMsgCh

	fmt.Println(len(msg.Data()))
}

func NewTestNode(t *testing.T, ctx context.Context) (*net.Node, chan net.Message) {
	logging.SetTestHook()
	receiveMsgCh := make(chan net.Message)

	cfg := medlet.DefaultConfig()
	cfg.Global.Datadir = testutil.TempDir(t)

	node, err := net.NewNode(ctx, cfg, receiveMsgCh)
	require.NoError(t, err)
	require.NoError(t, node.Start()) // start net service

	return node, receiveMsgCh
}

func cleanup(t *testing.T) {
	require.NoError(t, os.RemoveAll("testdata"))
}
