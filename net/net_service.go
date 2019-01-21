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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/crypto/hash"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// MedService service for medibloc p2p network
type MedService struct {
	ctx context.Context

	node       *Node
	dispatcher *Dispatcher
}

// NewNetService create netService
func NewNetService(cfg *medletpb.Config) (*MedService, error) {
	if networkConf := cfg.GetNetwork(); networkConf == nil {
		logging.Console().Error("config.conf should have network")
		return nil, ErrConfigLackNetWork
	}

	dispatcher := NewDispatcher()
	node, err := NewNode(cfg, dispatcher.receivedMessageCh)
	if err != nil {
		return nil, err
	}

	ms := &MedService{
		node:       node,
		dispatcher: dispatcher,
	}
	return ms, nil
}

// Node return the peer host
func (ms *MedService) Node() *Node {
	return ms.node
}

// Start start p2p manager.
func (ms *MedService) Start(ctx context.Context) error {
	logging.Console().Info("Starting MedService...")
	ms.ctx = ctx

	// start dispatcher.
	ms.dispatcher.Start(ctx)

	// start host.
	if err := ms.node.Start(ctx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to start MedService.")
		return err
	}

	logging.Console().Info("Started MedService.")

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			logging.Console().Info("Stopping MedService...")
		}
	}(ms.ctx)

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

// SendMessageToPeer send message to a peer.
func (ms *MedService) SendMessageToPeer(msgType string, data []byte, priority int, peerID string) {
	ms.node.SendMessageToPeer(msgType, data, priority, peerID)
}

// SendMessageToPeers send message to peers.
func (ms *MedService) SendMessageToPeers(msgType string, data []byte, priority int, filter PeerFilterAlgorithm) []string {
	return ms.node.SendMessageToPeers(msgType, data, priority, filter)
}

// Broadcast message.
func (ms *MedService) Broadcast(msgType string, data []byte, priority int) {
	ms.node.BroadcastMessage(msgType, data, priority)
}

// ClosePeer close connection
func (ms *MedService) ClosePeer(peerID string, reason error) {
	ms.node.ClosePeer(peerID, reason)
}

// SendPbMessageToPeer send protobuf message to peer
func (ms *MedService) SendPbMessageToPeer(msgType string, pb proto.Message, priority int, peerID string) {
	b, _ := proto.Marshal(pb)
	ms.node.SendMessageToPeer(msgType, b, priority, peerID)
}

// SendPbMessageToPeers send protobuf messages to filtered peers
func (ms *MedService) SendPbMessageToPeers(msgType string, pb proto.Message, priority int, filter PeerFilterAlgorithm) []string {
	b, _ := proto.Marshal(pb)
	return ms.SendMessageToPeers(msgType, b, priority, filter)
}

// RequestAndResponse set id to query and send to peers
func (ms *MedService) RequestAndResponse(ctx context.Context, query Query, f MessageCallback, filter PeerFilterAlgorithm) (bool, []error) {
	idBytes := hash.Sha3256([]byte(query.MessageType()), query.Hash(), byteutils.FromInt64(time.Now().UnixNano()))
	id := byteutils.Bytes2Hex(idBytes)
	query.SetID(id)

	responseCh := make(chan Message, 256)
	ms.Register(NewSubscriber("", responseCh, true, id, MessageWeightZero))
	defer ms.Deregister(NewSubscriber("", responseCh, true, id, MessageWeightZero))

	peers := ms.SendPbMessageToPeers(query.MessageType(), query.ProtoBuf(), MessagePriorityHigh, filter)
	errs := make([]error, len(peers))
	for i := 0; i < len(peers); i++ {
		select {
		case <-ctx.Done():
			errs[i] = ErrContextDone
			break
		case msg := <-responseCh:
			errs[i] = f(query, msg)
			if errs[i] == nil {
				return true, errs
			}
		}
	}
	return false, errs
}
