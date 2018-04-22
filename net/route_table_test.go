package net

import (
	"testing"

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
			nodeTestManager := NewNodeTestManager(tt.nodeNum, tt.seedNum)
			_, err := nodeTestManager.MakeNewTestNodes()
			assert.Nil(t, err)
			nodeTestManager.StartTestNodes()
			nodeTestManager.WaitRouteTableSync()

			// test whether route table peer list is correct
			for i := 0; i < tt.nodeNum; i++ {
				node, err := nodeTestManager.Node(i)
				assert.Nil(t, err)
				got := node.routeTable.peerStore.Peers()
				want := nodeTestManager.nodeIDs
				assert.Subset(t, got, want)
				assert.Subset(t, want, got)
			}

			nodeTestManager.StopTestNodes()
		})
	}
}
