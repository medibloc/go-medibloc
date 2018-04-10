package net

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/stretchr/testify/assert"
)

type msgFields struct {
	data     []byte
	fromIdx  int
	msgName  string
	priority int
	toIdx    int
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
				msgFields{
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
				msgFields{
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
				msgFields{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					1,
				},
				msgFields{
					[]byte{0x01},
					0,
					PongMessage,
					MessagePriorityNormal,
					1,
				},
				msgFields{
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
				msgFields{
					[]byte{0x00},
					0,
					PingMessage,
					MessagePriorityHigh,
					1,
				},
				msgFields{
					[]byte{0x00},
					1,
					PingMessage,
					MessagePriorityNormal,
					2,
				},
				msgFields{
					[]byte{0x01},
					1,
					PongMessage,
					MessagePriorityHigh,
					0,
				},
				msgFields{
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
			logging.Console().Info(fmt.Sprintf("MedService Test %s Start...", tt.name))

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
			medServiceTestManager.WaitRouteTableSync()

			// send messages
			for _, f := range tt.msgFields {
				medServices[f.fromIdx].SendMessageToPeer(f.msgName, f.data, f.priority, medServices[f.toIdx].node.ID())
			}

			timer := time.NewTimer(MessageCheckTimeout)
			cases := make([]reflect.SelectCase, len(messageChs)+1)
			for i, ch := range messageChs {
				cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
			}
			cases[len(messageChs)] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(timer.C)}

		waitMessageLoop:
			for receivedTotalCount < expectedTotalCount {
				chosen, value, ok := reflect.Select(cases)
				if !ok {
					if chosen == len(messageChs) {
						logging.Console().Info(fmt.Sprintf("Timeout"))
						break waitMessageLoop
					}
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

			medServiceTestManager.StopMedServices()

			logging.Console().Info(fmt.Sprintf("MedService Test %s Finished", tt.name))
		})
	}
}
