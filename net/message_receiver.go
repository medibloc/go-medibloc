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
	"container/list"
	"context"
	"io"
	"io/ioutil"
	"sync/atomic"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	"github.com/medibloc/go-medibloc/util/logging"
)

const MaxReadConcurrency = 10

type messageReceiver struct {
	context context.Context

	streamQ   *streamQ
	receiving int32
	streamCh  chan inet.Stream

	recvMessageCh chan<- Message
	doneCh        chan error
	bloomFilter   *BloomFilter
}

func newMessageReceiver(ctx context.Context, recvMessageCh chan<- Message, bf *BloomFilter) *messageReceiver {
	return &messageReceiver{
		context:       ctx,
		streamQ:       newStreamQ(),
		receiving:     0,
		streamCh:      make(chan inet.Stream),
		recvMessageCh: recvMessageCh,
		doneCh:        make(chan error),
		bloomFilter:   bf,
	}
}

func (mr *messageReceiver) Start() {
	logging.Console().Info("Start Message Receiver")
	go mr.loop()
}

func (mr *messageReceiver) loop() {
	for {
		select {
		case <-mr.context.Done():
			logging.Console().Info("Stop Message Receiver")
			return
		case s := <-mr.streamCh:
			mr.handelNewStream(s)
		case err := <-mr.doneCh:
			mr.finishReadMessage(err)
		}
	}
}

func (mr *messageReceiver) handelNewStream(s inet.Stream) {
	// writing concurrency is fulfilled. put message receiver queue
	if atomic.LoadInt32(&mr.receiving) >= MaxReadConcurrency {
		logging.Console().Debug("Read concurrency is full")
		mr.streamQ.put(s)
		return
	}
	atomic.AddInt32(&mr.receiving, 1)
	go mr.readMessage(s)
}

func (mr *messageReceiver) readMessage(s inet.Stream) {
	defer s.Reset()
	sender := s.Conn().RemotePeer()

	s.SetReadDeadline(time.Now().Add(StreamTTL))

	r := io.LimitReader(s, MaxMessageSize)

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		mr.doneCh <- err
		return
	}
	msg, err := newRecvMessage(bytes, sender)
	if err != nil {
		logging.Console().Debugf("Error unmarshaling data: %s", err)
		mr.doneCh <- err
		return
	}

	mr.bloomFilter.RecordRecvMessage(msg)
	mr.recvMessageCh <- msg
	mr.doneCh <- nil
}

func (mr *messageReceiver) finishReadMessage(err error) {
	s := mr.streamQ.pop()
	if s == nil { // no message in Queue
		atomic.AddInt32(&mr.receiving, -1)
		return
	}
	go mr.readMessage(s)
}

type streamQ struct {
	highPriorityStreams   *list.List
	normalPriorityStreams *list.List
}

func newStreamQ() *streamQ {
	return &streamQ{
		highPriorityStreams:   list.New(),
		normalPriorityStreams: list.New(),
	}
}

func (q *streamQ) pop() inet.Stream {
	out := q.highPriorityStreams.Front()
	if out != nil {
		return q.highPriorityStreams.Remove(out).(inet.Stream)
	}
	out = q.normalPriorityStreams.Front()
	if out != nil {
		return q.normalPriorityStreams.Remove(out).(inet.Stream)
	}
	return nil
}

func (q *streamQ) put(s inet.Stream) {
	//switch s.priority {
	//case MessagePriorityHigh:
	//	q.highPriorityStreams.PushBack(s)
	//default:
	//	q.normalPriorityStreams.PushBack(s)
	//}
	q.normalPriorityStreams.PushBack(s)
}
