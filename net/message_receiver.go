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
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type messageReceiver struct {
	context context.Context

	streamQ      *streamQ
	receiving    int32
	maxReceiving int32
	streamCh     chan inet.Stream

	recvMessageCh chan<- Message
	doneCh        chan error
	bloomFilter   *BloomFilter
}

func newMessageReceiver(ctx context.Context, recvMessageCh chan<- Message, bf *BloomFilter, maxConcurrency int32) *messageReceiver {
	if maxConcurrency == 0 {
		maxConcurrency = DefaultMaxReadConcurrency
	}
	return &messageReceiver{
		context:       ctx,
		streamQ:       newStreamQ(),
		receiving:     0,
		maxReceiving:  maxConcurrency,
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
			mr.handleNewStream(s)
		case err := <-mr.doneCh:
			mr.finishReadMessage(err)
		}
	}
}

func (mr *messageReceiver) handleNewStream(s inet.Stream) {
	// writing concurrency is fulfilled. put message receiver queue
	if mr.receiving >= mr.maxReceiving {
		logging.Console().Debug("Read concurrency is full")
		mr.streamQ.put(s)
		return
	}
	mr.receiving++
	go mr.readMessage(s)
}

func (mr *messageReceiver) readMessage(s inet.Stream) {
	defer s.Reset()
	sender := s.Conn().RemotePeer()

	err := s.SetReadDeadline(time.Now().Add(StreamTTL))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Debug("failed to set read deadline")
		mr.doneCh <- err
		return
	}

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
		mr.receiving--
		return
	}
	go mr.readMessage(s)
}

type streamQ struct {
	streams *list.List
}

func newStreamQ() *streamQ {
	return &streamQ{
		streams: list.New(),
	}
}

func (q *streamQ) pop() inet.Stream {
	out := q.streams.Front()
	if out != nil {
		return q.streams.Remove(out).(inet.Stream)
	}
	return nil
}

func (q *streamQ) put(s inet.Stream) {
	q.streams.PushBack(s)
}
