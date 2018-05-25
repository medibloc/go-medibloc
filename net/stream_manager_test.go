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
	"bytes"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/medibloc/go-medibloc/util/logging"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

const (
	maxstreamnumber = 10
	reservednumber  = 2
)

var (
	// weight of each msg type
	msgWeight = map[string]MessageWeight{
		HELLO:      MessageWeightZero,
		OK:         MessageWeightZero,
		BYE:        MessageWeightZero,
		SYNCROUTE:  MessageWeightZero,
		ROUTETABLE: MessageWeightRouteTable,

		SyncMetaRequest:       MessageWeightZero,
		SyncMeta:              MessageWeightChainChunks,
		SyncBlockChunkRequest: MessageWeightZero,
		SyncBlockChunk:        MessageWeightChainChunkData,

		"newblock": MessageWeightNewBlock,
		"dlblock":  MessageWeightZero,
		"dlreply":  MessageWeightZero,
		"newtx":    MessageWeightZero,
	}

	MsgTypes []string
)

func TestAllMsg(t *testing.T) {
	msgtypes := []string{HELLO, OK, BYE, SYNCROUTE, ROUTETABLE,
		SyncMetaRequest, SyncMeta, SyncBlockChunkRequest, SyncBlockChunk,
		"newblock", "dlblock", "dlreply", "newtx",
	}

	MsgTypes = append(MsgTypes, msgtypes...)
	run(t)
}

func TestUnvaluedMsg(t *testing.T) {
	msgtypes := []string{HELLO, OK, BYE, SYNCROUTE,
		SyncMetaRequest, SyncBlockChunkRequest,
		"dlblock", "dlreply", "newtx",
	}

	MsgTypes = append(MsgTypes, msgtypes...)
	run(t)
}

func run(t *testing.T) {

	cleanupTicker := time.NewTicker(CleanupInterval / 12)
	stopTicker := time.NewTicker(CleanupInterval / 12)
	times := 0
	for {
		select {
		case <-stopTicker.C:
			cleanupTicker.Stop()
			stopTicker.Stop()
			return
		case <-cleanupTicker.C:
			times++
			t.Logf("mock %d\n: max num = %d, reserved = %d\n", times, maxstreamnumber, reservednumber)
			sm := NewStreamManager()
			sm.fillMockStreams(t, maxstreamnumber)
			cleanup(t, sm)
		}
	}
}

func cleanup(t *testing.T, sm *StreamManager) {
	if sm.activePeersCount < maxstreamnumber {
		logging.Console().WithFields(logrus.Fields{
			"maxNum":      maxstreamnumber,
			"reservedNum": reservednumber,
			"currentNum":  sm.activePeersCount,
		}).Debug("No need for streams cleanup.")
		return
	}

	// total number of each msg type
	msgTotal := make(map[string]int)

	svs := make(StreamValueSlice, 0)
	sm.allStreams.Range(func(key, value interface{}) bool {
		stream := value.(*Stream)

		// t type, c count
		for t, c := range stream.msgCount {
			msgTotal[t] += c
		}

		svs = append(svs, &StreamValue{
			stream: stream,
		})

		return true
	})

	t.Log("total:")
	t.Log(orderedString(&msgTotal))

	for _, sv := range svs {
		for t, c := range sv.stream.msgCount {
			w, _ := msgWeight[t]
			sv.value += float64(c) * float64(w) / float64(msgTotal[t])
		}
	}

	sort.Sort(sort.Reverse(svs))

	t.Log("sorted:")
	for _, sv := range svs {
		t.Log(strconv.FormatFloat(sv.value, 'f', 3, 64), orderedString(&sv.stream.msgCount))
	}

	t.Log("eliminated:")
	eliminated := svs[maxstreamnumber-reservednumber:]
	for _, sv := range eliminated {
		t.Log(strconv.FormatFloat(sv.value, 'f', 3, 64), orderedString(&sv.stream.msgCount))
	}
	t.Log("")
}

func (sm *StreamManager) fillMockStreams(t *testing.T, num int32) {
	sm.activePeersCount = num
	t.Log("details:")

	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9999")

	for ; num > 0; num-- {
		key := "s" + strconv.FormatInt(int64(num), 10)

		rand.Seed(time.Now().UnixNano())
		msgCount := make(map[string]int)
		for _, t := range MsgTypes {
			msgCount[t] = rand.Intn(50)
		}

		pid, _ := peer.IDFromString(key)
		sm.allStreams.Store(key, &Stream{
			pid:      pid,
			addr:     addr,
			msgCount: msgCount,
		})

		t.Log(key, orderedString(&msgCount))
	}
}

func orderedString(mc *map[string]int) string {
	var buffer bytes.Buffer
	for _, t := range MsgTypes {
		buffer.WriteString(t + ":" + strconv.Itoa((*mc)[t]) + " ")
	}
	return buffer.String()
}
