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
	"testing"

	"github.com/medibloc/go-medibloc/util/logging"

	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	TestMessageType = "TEST"
	TestMessageData = []byte("TESTDATA")
)

func TestNewMedService(t *testing.T) {
	//logging.SetTestHook()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer cleanup(t)

	msgCh := make(chan net.Message)

	seed, seedPI := NewTestNetService(t, ctx)
	seed.Register(net.NewSubscriber("TEST", msgCh, true, TestMessageType, net.MessagePriorityNormal))

	seed.Node().ConnManager()

	ns, _ := NewTestNetService(t, ctx)

	ns.Node().Peerstore().AddAddrs(seedPI.ID, seedPI.Addrs, peerstore.PermanentAddrTTL)
	ns.SendMessageToPeer(TestMessageType, TestMessageData, net.MessagePriorityNormal, seedPI.ID.Pretty())

	receivedMsg := <-msgCh

	assert.Equal(t, ns.Node().ID().Pretty(), receivedMsg.MessageFrom())
	assert.Equal(t, TestMessageType, receivedMsg.MessageType())
	assert.True(t, byteutils.Equal(TestMessageData, receivedMsg.Data()))

	t.Log("received data: ", string(receivedMsg.Data()))

}

func NewTestNetService(t *testing.T, ctx context.Context) (*net.MedService, peerstore.PeerInfo) {
	logging.SetTestHook()
	cfg := medlet.DefaultConfig()
	cfg.Global.Datadir = testutil.TempDir(t)

	ns, err := net.NewMedService(ctx, cfg)
	require.NoError(t, err)
	require.NoError(t, ns.Start()) // start net service

	pi := peerstore.PeerInfo{
		ID:    ns.Node().ID(),
		Addrs: ns.Node().Addrs(),
	}
	t.Log("new net service info: ", pi)

	return ns, pi
}
