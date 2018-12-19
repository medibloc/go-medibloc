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
	"context"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/medibloc/go-medibloc/metrics"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

var (
	metricsDispatcherCached     = metrics.NewGauge("med.net.dispatcher.cached")
	metricsDispatcherDuplicated = metrics.NewMeter("med.net.dispatcher.duplicated")
)

// Dispatcher a message dispatcher service.
type Dispatcher struct {
	subscribersMap     *sync.Map
	context            context.Context
	receivedMessageCh  chan Message
	dispatchedMessages *lru.Cache
	filters            *sync.Map
}

// ReceivedMessageCh return dispatcher's received message channel
func (dp *Dispatcher) ReceivedMessageCh() chan<- Message {
	return dp.receivedMessageCh
}

// NewDispatcher create Dispatcher instance.
func NewDispatcher(ctx context.Context) *Dispatcher {
	dispatchedMessages, _ := lru.New(51200)

	return &Dispatcher{
		subscribersMap:     new(sync.Map),
		context:            ctx,
		receivedMessageCh:  make(chan Message, 65536),
		filters:            new(sync.Map),
		dispatchedMessages: dispatchedMessages,
	}
}

// Register register subscribers.
func (dp *Dispatcher) Register(subscribers ...*Subscriber) {
	for _, v := range subscribers {
		mt := v.MessageType()
		m, _ := dp.subscribersMap.LoadOrStore(mt, new(sync.Map))
		m.(*sync.Map).Store(v, true)
		dp.filters.Store(mt, v.DoFilter())
	}
}

// Deregister deregister subscribers.
func (dp *Dispatcher) Deregister(subscribers ...*Subscriber) {
	for _, v := range subscribers {
		mt := v.MessageType()
		m, _ := dp.subscribersMap.Load(mt)
		if m == nil {
			continue
		}
		m.(*sync.Map).Delete(v)
		dp.filters.Delete(mt)
	}
}

// Start start message dispatch goroutine.
func (dp *Dispatcher) Start() {
	logging.Console().Info("Starting MedService Dispatcher...")
	go dp.loop()
}

func (dp *Dispatcher) loop() {
	logging.Console().Info("Started NewService Dispatcher.")

	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			metricsDispatcherCached.Update(int64(len(dp.receivedMessageCh)))
		case <-dp.context.Done():
			logging.Console().Info("Stopped MedService Dispatcher.")
			return
		case msg := <-dp.receivedMessageCh:
			if dp.isFiltered(msg) {
				continue
			}

			msgType := msg.MessageType()
			v, _ := dp.subscribersMap.Load(msgType)
			if v == nil {
				continue
			}
			m, _ := v.(*sync.Map)

			m.Range(func(key, value interface{}) bool {
				select {
				case key.(*Subscriber).msgChan <- msg:
				default:
					logging.WithFields(logrus.Fields{
						"msgType": msgType,
					}).Debug("timeout to dispatch message.")
				}
				return true
			})
		}
	}
}

// filter handle new message to chan, then subscribers will be notified to process.
func (dp *Dispatcher) isFiltered(msg Message) bool {
	hash := msg.Hash()
	mt := msg.MessageType()
	v, ok := dp.filters.Load(mt)

	if !ok {
		logging.WithFields(logrus.Fields{
			"from": msg.MessageFrom(),
			"type": msg.MessageType(),
		}).Warn("Unregistered message received.")
		return true
	}

	if v.(bool) {
		if exist, _ := dp.dispatchedMessages.ContainsOrAdd(hash, hash); exist == true {
			// duplicated message, ignore.
			metricsDuplicatedMessage(msg.MessageType())
			return true
		}
	}
	return false
}

func metricsDuplicatedMessage(messageName string) {
	metricsDispatcherDuplicated.Mark(int64(1))
	meter := metrics.NewMeter(fmt.Sprintf("med.net.dispatcher.duplicated.%s", messageName))
	meter.Mark(int64(1))
}
