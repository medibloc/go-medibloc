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
