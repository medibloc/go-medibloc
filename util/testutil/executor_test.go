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
	"reflect"
	"testing"

	"time"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/net"
	"github.com/stretchr/testify/require"
)

func TestNetworkUtil(t *testing.T) {
	nt := NewNetwork(t, 3)
	nt.NewSeedNode()
	for i := 0; i < 2; i++ {
		nt.NewNode()
	}
	nt.Start()
	defer nt.Cleanup()
	nt.WaitForEstablished()

	genesis, err := nt.Nodes[0].med.BlockManager().BlockByHeight(core.GenesisHeight)
	require.NoError(t, err)
	block := NewTestBlock(t, genesis)

	from := nt.Nodes[0].med.NetService()
	from.Broadcast(core.MessageTypeNewBlock, block, 1)

	to := nt.Nodes[1]
	ch := make(chan net.Message)
	subscriber := net.NewSubscriber(to, ch, false, core.MessageTypeNewBlock, 1)
	to.med.NetService().Register(subscriber)
	defer to.med.NetService().Deregister(subscriber)
	msg := <-ch

	bd, err := core.BytesToBlockData(msg.Data())
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(block.GetBlockData(), bd))
}

func TestNetworkMiner(t *testing.T) {
	nt := NewNetwork(t, 3)
	seed := nt.NewSeedNode()
	nt.SetMinerFromDynasties(seed)
	for i := 0; i < 2; i++ {
		node := nt.NewNode()
		nt.SetMinerFromDynasties(node)
	}
	nt.Start()
	defer nt.Cleanup()
	nt.WaitForEstablished()

	waitTime := dpos.BlockInterval + time.Second
	timer := time.NewTimer(waitTime)
	defer timer.Stop()

	for _, node := range nt.Nodes {
		for {
			height := node.med.BlockManager().TailBlock().Height()
			if height >= 2 {
				break
			}

			select {
			case <-timer.C:
				require.True(t, false, "Block Mining Timeout")
			default:
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
