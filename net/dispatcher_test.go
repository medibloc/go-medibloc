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
	"strings"
	"testing"
	"time"
)

// Message Type
var (
	PingMessage = "ping"
	PongMessage = "pong"
)

var (
	MessageCheckTimeout = 1 * time.Second
)

type baseMsgFields struct {
	msgName string
	msgType string
	from    string
	data    []byte
}

type msgTypes struct {
	msgType  string
	doFilter bool
}

func TestDispatcher_PutMessage(t *testing.T) {
	tests := []struct {
		name           string
		messageCh      chan Message
		msgTypes       []msgTypes
		msgFieldsArr   []baseMsgFields
		expectedCounts []int
	}{
		{
			"PutMessageTest1",
			make(chan Message, 8),
			[]msgTypes{
				{
					PingMessage,
					false,
				},
			},
			[]baseMsgFields{
				{
					"FirstPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
			},
			[]int{1},
		},
		{
			"PutMessageTest2",
			make(chan Message, 8),
			[]msgTypes{
				{
					PingMessage,
					false,
				},
			},
			[]baseMsgFields{
				{
					"FirstPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
				{
					"FirstPong",
					PongMessage,
					"from",
					[]byte{0x01},
				},
			},
			[]int{1},
		},
		{
			"PutMessageTest3",
			make(chan Message, 8),
			[]msgTypes{
				{
					PingMessage,
					false,
				},
			},
			[]baseMsgFields{
				{
					"FirstPong",
					PongMessage,
					"from",
					[]byte{0x01},
				},
				{
					"FirstPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
			},
			[]int{1},
		},
		{
			"PutMessageTest4",
			make(chan Message, 8),
			[]msgTypes{
				{
					PingMessage,
					false,
				},
			},
			[]baseMsgFields{
				{
					"FirstPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
				{
					"SecondPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
			},
			[]int{2},
		},
		{
			"PutMessageTest5",
			make(chan Message, 8),
			[]msgTypes{
				{
					PingMessage,
					true,
				},
			},
			[]baseMsgFields{
				{
					"FirstPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
				{
					"SecondPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
			},
			[]int{1},
		},
		{
			"PutMessageTest6",
			make(chan Message, 8),
			[]msgTypes{
				{
					PingMessage,
					true,
				},
				{
					PongMessage,
					false,
				},
			},
			[]baseMsgFields{
				{
					"FirstPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
				{
					"FirstPong",
					PongMessage,
					"from",
					[]byte{0x01},
				},
				{
					"SecondPing",
					PingMessage,
					"from",
					[]byte{0x00},
				},
				{
					"SecondPong",
					PongMessage,
					"from",
					[]byte{0x01},
				},
			},
			[]int{1, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedCounts := make([]int, len(tt.msgTypes))
			receivedTotalCount := 0
			expectedTotalCount := 0
			for _, v := range tt.expectedCounts {
				expectedTotalCount += v
			}

			// make new dispatcher
			dp := makeNewTestDispatcher()
			// register msgTypes
			for _, msgType := range tt.msgTypes {
				dp.Register(NewSubscriber(t, tt.messageCh, msgType.doFilter, msgType.msgType, MessageWeightZero))
			}

			// start dispatcher
			dp.Start()

			// put messages
			for _, msgFields := range tt.msgFieldsArr {
				dp.PutMessage(NewBaseMessage(msgFields.msgType, msgFields.from, msgFields.data))
			}

			timer := time.NewTimer(MessageCheckTimeout)
		waitMessageLoop:
			for receivedTotalCount < expectedTotalCount {
				select {
				case receivedMsg := <-tt.messageCh:
					// find index of msgTypes
					idx := 0
					for i, v := range tt.msgTypes {
						if strings.Compare(receivedMsg.MessageType(), v.msgType) == 0 {
							idx = i
							break
						}
					}

					receivedCounts[idx]++
					receivedTotalCount++
				case <-timer.C:
					t.Log("Timeout")
					break waitMessageLoop
				}
			}

			// compare expectedCounts and receivedCounts
			if !reflect.DeepEqual(tt.expectedCounts, receivedCounts) {
				t.Errorf("receivedCounts() = %v, want %v", receivedCounts, tt.expectedCounts)
			}

			// stop dispatcher
			dp.Stop()
		})
	}
}
