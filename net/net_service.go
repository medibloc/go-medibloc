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

	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// MedService service for medibloc p2p network
type MedService struct {
	context context.Context

	node       *Node
	dispatcher *Dispatcher
}

// NewMedService create netService
func NewMedService(ctx context.Context, cfg *medletpb.Config) (*MedService, error) {
	if networkConf := cfg.GetNetwork(); networkConf == nil {
		logging.Console().Error("config.conf should have network")
		return nil, ErrConfigLackNetWork
	}

	dispatcher := NewDispatcher(ctx)
	node, err := NewNode(ctx, cfg, dispatcher.receivedMessageCh)
	if err != nil {
		return nil, err
	}

	ms := &MedService{
		node:       node,
		context:    ctx,
		dispatcher: dispatcher,
	}
	return ms, nil
}

// Node return the peer host
func (ms *MedService) Node() *Node {
	return ms.node
}

// Start start p2p manager.
func (ms *MedService) Start() error {
	logging.Console().Info("Starting MedService...")

	// start dispatcher.
	ms.dispatcher.Start()

	// start host.
	if err := ms.node.Start(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed receiver start MedService.")
		return err
	}

	logging.Console().Info("Started MedService.")

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			logging.Console().Info("Stopping MedService...")
		}
	}(ms.context)

	return nil
}

// Register register the subscribers.
func (ms *MedService) Register(subscribers ...*Subscriber) {
	ms.dispatcher.Register(subscribers...)
}

// Deregister Deregister the subscribers.
func (ms *MedService) Deregister(subscribers ...*Subscriber) {
	ms.dispatcher.Deregister(subscribers...)
}

// Relay message.
//func (ms *MedService) Relay(name string, msg Serializable, priority int) {
//	ms.host.RelayMessage(name, msg, priority)
//}

// SendMessageToPeer send message receiver a peer.
func (ms *MedService) SendMessageToPeer(msgType string, data []byte, priority int, peerID string) {
	ms.node.SendMessageToPeer(msgType, data, priority, peerID)
}

// SendMessageToPeers send message receiver peers.
func (ms *MedService) SendMessageToPeers(msgType string, data []byte, priority int, filter PeerFilterAlgorithm) []string {
	return ms.node.SendMessageToPeers(msgType, data, priority, filter)
}

// Broadcast message.
func (ms *MedService) Broadcast(msgType string, data []byte, priority int) {
	ms.node.BroadcastMessage(msgType, data, priority)
}

func (ms *MedService) ClosePeer(peerID string, reason error) {
	ms.node.ClosePeer(peerID, reason)
}
