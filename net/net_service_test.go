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
	"reflect"
	"testing"
	"time"

	"errors"

	proto "github.com/gogo/protobuf/proto"
	nettestpb "github.com/medibloc/go-medibloc/net/testpb"
	"github.com/stretchr/testify/assert"
)

var (
	BroadcastMsgTestCheckTimeout = 10 * time.Second
	SendMsgTestCheckTimeout      = 5 * time.Second

	ErrInvalidProtoToTestMsg = errors.New("protobuf message cannot be converted into TestMsg")
)

type msgFields struct {
	data     []byte
	fromIdx  int
	msgName  string
	priority int
	toIdx    int
}

type TestMsg struct {
	data []byte
}

func (stm *TestMsg) ToProto() (proto.Message, error) {
	return &nettestpb.TestMsg{
		Data: stm.data,
	}, nil
}
func (stm *TestMsg) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*nettestpb.TestMsg); ok {
		if msg != nil {
			stm.data = msg.Data
		}
	}
	return ErrInvalidProtoToTestMsg
}

func TestNetService_SendMessageToPeer(t *testing.T) {
	tests := []struct {
		expectedCounts [][]int
		name           string
		nodeNum        int
		msgFields      []msgFields
		msgTypes       []string
	}{
		{
			[][]int{
				{0},
				{1},
			},
			"SendMessage1",
			2,
			[]msgFields{
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					1,
				},
			},
			[]string{
				PingMessage,
			},
		},
		{
			[][]int{
				{0},
				{2},
			},
			"SendMessage2",
			2,
			[]msgFields{
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					1,
				},
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					1,
				},
			},
			[]string{
				PingMessage,
			},
		},
		{
			[][]int{
				{1, 0},
				{1, 1},
			},
			"SendMessage3",
			2,
			[]msgFields{
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					1,
				},
				{
					[]byte{0x01},
					0,
					PongMessage,
					MessagePriorityNormal,
					1,
				},
				{
					[]byte{0x00},
					1,
					PingMessage,
					MessagePriorityHigh,
					0,
				},
			},
			[]string{
				PingMessage,
				PongMessage,
			},
		},
		{
			[][]int{
				{0, 1},
				{1, 0},
				{1, 1},
			},
			"SendMessage4",
			3,
			[]msgFields{
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					1,
				},
				{
					[]byte{0x00},
					1,
					PingMessage,
					MessagePriorityNormal,
					2,
				},
				{
					[]byte{0x01},
					1,
					PongMessage,
					MessagePriorityHigh,
					0,
				},
				{
					[]byte{0x01},
					0,
					PongMessage,
					MessagePriorityLow,
					2,
				},
			},
			[]string{
				PingMessage,
				PongMessage,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create message channels
			messageChs := make([]chan Message, tt.nodeNum)
			for i := 0; i < tt.nodeNum; i++ {
				messageChs[i] = make(chan Message, 8)
			}
			receivedCounts := make([][]int, tt.nodeNum)
			for i := 0; i < tt.nodeNum; i++ {
				receivedCounts[i] = make([]int, len(tt.msgTypes))
			}
			receivedTotalCount := 0
			expectedTotalCount := 0
			for _, r := range tt.expectedCounts {
				for _, v := range r {
					expectedTotalCount += v
				}
			}

			// create a medServiceTestManager
			medServiceTestManager := NewMedServiceTestManager(tt.nodeNum, 1)
			medServices, err := medServiceTestManager.MakeNewTestMedService()
			assert.Nil(t, err)
			if len(medServices) != tt.nodeNum {
				t.Errorf("Number of MedServices = %d, want %d", len(medServices), tt.nodeNum)
			}
			// register msgTypes to each test message channel
			for _, t := range tt.msgTypes {
				for idx, s := range medServices {
					s.dispatcher.Register(NewSubscriber(t, messageChs[idx], false, t, MessageWeightZero))
				}
			}
			medServiceTestManager.StartMedServices()
			defer medServiceTestManager.StopMedServices()
			medServiceTestManager.WaitRouteTableSync()
			medServiceTestManager.WaitStreamReady()

			// send messages
			for _, f := range tt.msgFields {
				medServices[f.fromIdx].SendMessageToPeer(f.msgName, f.data, f.priority, medServices[f.toIdx].node.ID())
			}

			timer := time.NewTimer(SendMsgTestCheckTimeout)
			cases := make([]reflect.SelectCase, len(messageChs)+1)
			for i, ch := range messageChs {
				cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
			}
			cases[len(messageChs)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(timer.C)}

		waitMessageLoop:
			for receivedTotalCount < expectedTotalCount {
				chosen, value, ok := reflect.Select(cases)

				if chosen == len(messageChs) {
					t.Logf("Timeout with expectedTotalCount: %d, receivedTotalCount: %d", +expectedTotalCount, receivedTotalCount)
					break waitMessageLoop
				}

				if !ok {
					continue
				}

				// find index of msgTypes
				idx := 0
				for i, v := range tt.msgTypes {
					if value.Interface().(*BaseMessage).MessageType() == v {
						idx = i
						break
					}
				}

				receivedCounts[chosen][idx]++
				receivedTotalCount++
			}

			// compare expectedCounts and receivedCounts
			if !reflect.DeepEqual(tt.expectedCounts, receivedCounts) {
				t.Errorf("receivedCounts() = %v, want %v", receivedCounts, tt.expectedCounts)
			}
		})
	}
}

func TestNetService_SendBroadcast(t *testing.T) {
	tests := []struct {
		expectedCounts [][]int
		name           string
		nodeNum        int
		msgFields      []msgFields
		msgTypes       []string
	}{
		{
			[][]int{
				{0},
				{1},
			},
			"BroadCastMessage1",
			2,
			[]msgFields{
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					-1,
				},
			},
			[]string{
				PingMessage,
			},
		},
		{
			[][]int{
				{1},
				{1},
				{2},
			},
			"BroadCastMessage2",
			3,
			[]msgFields{
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					-1,
				},
				{
					[]byte{0x01},
					1,
					PingMessage,
					MessagePriorityHigh,
					-1,
				},
			},
			[]string{
				PingMessage,
			},
		},
		{
			[][]int{
				{1, 1},
				{2, 0},
				{1, 1},
			},
			"BroadCastMessage3",
			3,
			[]msgFields{
				{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					-1,
				},
				{
					[]byte{0x01},
					1,
					PongMessage,
					MessagePriorityNormal,
					-1,
				},
				{
					[]byte{0x02},
					2,
					PingMessage,
					MessagePriorityHigh,
					-1,
				},
			},
			[]string{
				PingMessage,
				PongMessage,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create message channels
			messageChs := make([]chan Message, tt.nodeNum)
			for i := 0; i < tt.nodeNum; i++ {
				messageChs[i] = make(chan Message, 8)
			}
			receivedCounts := make([][]int, tt.nodeNum)
			for i := 0; i < tt.nodeNum; i++ {
				receivedCounts[i] = make([]int, len(tt.msgTypes))
			}
			receivedTotalCount := 0
			expectedTotalCount := 0
			for _, r := range tt.expectedCounts {
				for _, v := range r {
					expectedTotalCount += v
				}
			}

			// create a medServiceTestManager
			medServiceTestManager := NewMedServiceTestManager(tt.nodeNum, 1)
			medServices, err := medServiceTestManager.MakeNewTestMedService()
			assert.Nil(t, err)
			if len(medServices) != tt.nodeNum {
				t.Errorf("Number of MedServices = %d, want %d", len(medServices), tt.nodeNum)
			}
			// register msgTypes to each test message channel
			for _, t := range tt.msgTypes {
				for idx, s := range medServices {
					s.dispatcher.Register(NewSubscriber(t, messageChs[idx], false, t, MessageWeightZero))
				}
			}
			medServiceTestManager.StartMedServices()
			defer medServiceTestManager.StopMedServices()
			medServiceTestManager.WaitRouteTableSync()
			medServiceTestManager.WaitStreamReady()

			// send messages
			for _, f := range tt.msgFields {
				medServices[f.fromIdx].Broadcast(f.msgName, &TestMsg{data: f.data}, f.priority)
			}

			timer := time.NewTimer(BroadcastMsgTestCheckTimeout)
			cases := make([]reflect.SelectCase, len(messageChs)+1)
			for i, ch := range messageChs {
				cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
			}
			cases[len(messageChs)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(timer.C)}

		waitMessageLoop:
			for receivedTotalCount < expectedTotalCount {
				chosen, value, ok := reflect.Select(cases)

				if chosen == len(messageChs) {
					t.Logf("Timeout with expectedTotalCount: %d, receivedTotalCount: %d", +expectedTotalCount, receivedTotalCount)
					break waitMessageLoop
				}

				if !ok {
					continue
				}

				// find index of msgTypes
				idx := 0
				for i, v := range tt.msgTypes {
					if value.Interface().(*BaseMessage).MessageType() == v {
						idx = i
						break
					}
				}

				receivedCounts[chosen][idx]++
				receivedTotalCount++
			}

			// compare expectedCounts and receivedCounts
			if !reflect.DeepEqual(tt.expectedCounts, receivedCounts) {
				t.Errorf("receivedCounts() = %v, want %v", receivedCounts, tt.expectedCounts)
			}
		})
	}
}
