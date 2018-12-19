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
	"time"

	host "github.com/libp2p/go-libp2p-host"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type messageSender struct {
	host    host.Host
	context context.Context

	messageQ   *sendMessageQ
	sending    int32
	maxSending int32

	msgCh  chan *SendMessage
	doneCh chan error
}

func newMessageSender(ctx context.Context, host host.Host, maxConcurrency int32) *messageSender {
	if maxConcurrency == 0 {
		maxConcurrency = DefaultMaxWriteConcurrency
	}
	return &messageSender{
		host:       host,
		context:    ctx,
		messageQ:   newMessageQ(),
		sending:    0,
		maxSending: maxConcurrency,
		msgCh:      make(chan *SendMessage),
		doneCh:     make(chan error),
	}
}

func (ms *messageSender) Start() {
	logging.Console().Info("Start Message Sender")
	go ms.loop()
}

func (ms *messageSender) loop() {

	for {
		select {
		case <-ms.context.Done():
			logging.Console().Info("Stop Message Sender")
			return
		case msg := <-ms.msgCh:
			ms.handleNewMessage(msg)
		case err := <-ms.doneCh:
			ms.finishSendMessage(err)
		}
	}
}

func (ms *messageSender) handleNewMessage(msg *SendMessage) {
	// writing concurrency is fulfilled. put message to queue
	if ms.sending >= ms.maxSending {
		ms.messageQ.put(msg)
		return
	}
	ms.sending++
	go ms.writeMessage(msg)
}

func (ms *messageSender) writeMessage(msg *SendMessage) {
	ctx, cancel := context.WithTimeout(ms.context, StreamTTL)
	defer cancel()

	s, err := ms.host.NewStream(ctx, msg.receiver, MedProtocolID)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"to":  msg.receiver.Pretty(),
			"err": err,
		}).Error("failed to open stream for send msg")
		ms.doneCh <- err
		return
	}
	defer s.Close()

	err = s.SetWriteDeadline(time.Now().Add(StreamTTL))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Debug("failed to set write deadline")
		ms.doneCh <- err
		return
	}
	if n, err := s.Write(msg.Bytes()); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"to":           msg.receiver.Pretty(),
			"msgType":      msg.MessageType(),
			"msg":          msg.Data(),
			"remainLength": n,
			"err":          err,
		}).Error("failed to write msg on stream")
		ms.doneCh <- err
		return
	}
	ms.doneCh <- nil
}

func (ms *messageSender) finishSendMessage(err error) {
	msg := ms.messageQ.pop()
	if msg == nil { // no message in Queue
		ms.sending--
		return
	}
	go ms.writeMessage(msg)
}

type sendMessageQ struct {
	highPriorityMessages   *list.List
	normalPriorityMessages *list.List
	lowPriorityMessages    *list.List
}

func newMessageQ() *sendMessageQ {
	return &sendMessageQ{
		highPriorityMessages:   list.New(),
		normalPriorityMessages: list.New(),
		lowPriorityMessages:    list.New(),
	}
}

func (q *sendMessageQ) pop() *SendMessage {
	out := q.highPriorityMessages.Front()
	if out != nil {
		return q.highPriorityMessages.Remove(out).(*SendMessage)
	}
	out = q.normalPriorityMessages.Front()
	if out != nil {
		return q.normalPriorityMessages.Remove(out).(*SendMessage)
	}
	out = q.lowPriorityMessages.Front()
	if out != nil {
		return q.lowPriorityMessages.Remove(out).(*SendMessage)
	}
	return nil
}

func (q *sendMessageQ) put(msg *SendMessage) {
	switch msg.priority {
	case MessagePriorityHigh:
		q.highPriorityMessages.PushBack(msg)
	case MessagePriorityNormal:
		q.normalPriorityMessages.PushBack(msg)
	default:
		q.lowPriorityMessages.PushBack(msg)
	}
}
