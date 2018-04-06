package net

import (
	"testing"

	"time"

	"sync"

	"fmt"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/medibloc/go-medibloc/util/logging"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	RouteTableCheckInterval = 100 * time.Millisecond
)

func TestRouteTable_SyncWithPeer(t *testing.T) {
	tests := []struct {
		name    string
		nodeNum int
		seedNum int
	}{
		{
			"RoutTableSyncWithOneSeedNode",
			3,
			1,
		},
		{
			"RoutTableSyncWithTwoSeedNode",
			5,
			2,
		},
		{
			"RoutTableSyncWithThreeSeedNode",
			7,
			3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logging.Console().Info(fmt.Sprintf("Test %s with %d nodes Start...", tt.name, tt.nodeNum))
			var wg sync.WaitGroup

			nodeNum := tt.nodeNum
			seedNum := tt.seedNum
			nodeArr := make([]*Node, nodeNum)
			var err error
			var seedNodes []ma.Multiaddr
			var seedNodeIDs, allNodeIDs []peer.ID

			// make all test nodes
			for i := 0; i < nodeNum; i++ {
				nodeArr[i], err = makeNewTestNode("")
				assert.Nil(t, err)
			}

			// set value of seedNodes, seedNodeIDs
			for i := 0; i < seedNum; i++ {
				seedMultiaddrs, err := convertListenAddrToMultiAddr(nodeArr[i].config.Listen)
				assert.Nil(t, err)
				newSeedNodes, err := convertMultiAddrToIPFSMultiAddr(seedMultiaddrs, nodeArr[i].ID())
				assert.Nil(t, err)
				for _, v := range newSeedNodes {
					seedNodes = append(seedNodes, v)
				}
				seedNodeIDs = append(seedNodeIDs, nodeArr[i].id)
			}

			// set value of allNodeIDs
			for i := 0; i < nodeNum; i++ {
				allNodeIDs = append(allNodeIDs, nodeArr[i].id)
			}

			// setup seedNodes to every nodes
			for i := 0; i < nodeNum; i++ {
				nodeArr[i].routeTable.seedNodes = seedNodes
			}

			// start seed nodes
			for i := 0; i < seedNum; i++ {
				nodeArr[i].Start()
			}

			// wait while seed nodes boot
			time.Sleep(100 * time.Millisecond)

			// start normal nodes
			for i := nodeNum - 1; i >= seedNum; i-- {
				nodeArr[i].Start()
			}

			// wait for sync route table
			for i := 0; i < nodeNum; i++ {
				wg.Add(1)
				go waitRouteTableSyncLoop(&wg, nodeArr[i], allNodeIDs)
			}

			// wait until all nodes are synced
			logging.Console().Info("Waiting waitGroup Start...")
			wg.Wait()
			logging.Console().Info("Waiting waitGroup Finished")

			// test whether route table peer list is correct
			for i := 0; i < nodeNum; i++ {
				got := nodeArr[i].routeTable.peerStore.Peers()
				want := allNodeIDs
				assert.Subset(t, got, want)
				assert.Subset(t, want, got)
			}

			// for debug
			//for i := 0; i < nodeNum; i++ {
			//	nodeArr[i].routeTable.PrintPeers()
			//}

			// stop all nodes
			for i := 0; i < nodeNum; i++ {
				nodeArr[i].Stop()
			}

			logging.Console().Info(fmt.Sprintf("Test %s with %d nodes Finished", tt.name, tt.nodeNum))
		})
	}
}

func waitRouteTableSyncLoop(wg *sync.WaitGroup, node *Node, nodeIDs []peer.ID) {
	defer wg.Done()
	ids := make([]peer.ID, len(nodeIDs))
	copy(ids, nodeIDs)
	remainIteration := 100

	for len(ids) > 0 {
		i := 0
		for _, v := range ids {
			if node.routeTable.peerStore.Addrs(v) == nil {
				ids[i] = v
				i++
			}
		}
		ids = ids[:i]

		time.Sleep(RouteTableCheckInterval)

		remainIteration--
		// fail if (sec * len(nodeIDs)) seconds left
		if remainIteration < 1 && len(ids) > 0 {
			logging.Console().WithFields(logrus.Fields{
				"node ID":                          node.ID(),
				"routeTable not synced node count": len(ids),
			}).Warn("route table not synced in time")
			return
		}
	}
}
