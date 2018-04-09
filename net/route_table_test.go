package net

import (
	"testing"

	"time"

	"sync"

	"fmt"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/stretchr/testify/assert"
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
			var allNodeIDs []peer.ID

			nodeArr, allNodeIDs, err = makeAndSetNewTestNodes(nodeNum, seedNum)
			assert.Nil(t, err)

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
			//	PrintRouteTablePeers(nodeArr[i].routeTable)
			//}

			// stop all nodes
			for i := 0; i < nodeNum; i++ {
				nodeArr[i].Stop()
			}

			logging.Console().Info(fmt.Sprintf("Test %s with %d nodes Finished", tt.name, tt.nodeNum))
		})
	}
}
