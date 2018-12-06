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
	"errors"
	"time"

	"github.com/gogo/protobuf/proto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
)

// DefaultConnMgrHighWater is the default value for the connection managers
// values from "ipfs/go-ipfs-config"
const (
	DefaultConnMgrHighWater   = 900
	DefaultConnMgrLowWater    = 600
	DefaultConnMgrGracePeriod = time.Second * 20
)

// Network system parameters
const (
	MaxMessageSize = inet.MessageSizeMax
	StreamTTL      = 1024 * time.Second
)

// Message Priority.
const (
	MessagePriorityHigh = iota
	MessagePriorityNormal
	MessagePriorityLow
)

// Default Config
const (
	DefaultPrivateKeyFile  = "network.key"
	DefaultBootstrapPeriod = 30 * time.Second
	DefaultCacheFile       = "network.cache"
	DefaultCachePeriod     = 3 * time.Minute
)

// Protocols
const (
	MedProtocolID    = "/med/1.0.0"
	MedDHTProtocolID = "/med/kad/1.0.0"
)

// Error types
var (
	ErrPeerIsNotConnected = errors.New("peer is not connected")
)

// Message interface for message.
type Message interface {
	MessageType() string
	MessageFrom() string
	Data() []byte
	Hash() string
}

// Serializable model
type Serializable interface {
	ToProto() (proto.Message, error)
	FromProto(proto.Message) error
}

// PeerFilterAlgorithm is the algorithm used receiver filter peers
type PeerFilterAlgorithm interface {
	Filter([]peer.ID) []peer.ID
}

// Service net Service interface
type Service interface {
	Start() error

	Node() *Node

	// dispatcher
	Register(...*Subscriber)
	Deregister(...*Subscriber)

	// host
	SendMessageToPeer(msgType string, data []byte, priority int, peerID string)
	SendMessageToPeers(msgType string, data []byte, priority int, filter PeerFilterAlgorithm) []string
	Broadcast(msgType string, data []byte, priority int)
	//Relay(string, Serializable, int)
	ClosePeer(peerID string, reason error)
}

// MessageWeight float64
type MessageWeight float64

// const
const (
	MessageWeightZero = MessageWeight(0)
	MessageWeightNewTx
	MessageWeightNewBlock = MessageWeight(0.5)
	MessageWeightRouteTable
	MessageWeightChainChunks
	MessageWeightChainChunkData
)

// Subscriber subscriber.
type Subscriber struct {
	// id usually the owner/creator, used for troubleshooting .
	id interface{}

	// msgChan chan for subscribed message.
	msgChan chan Message

	// msgType message type receiver subscribe
	msgType string

	// msgWeight weight of msgType
	msgWeight MessageWeight

	// doFilter dup message
	doFilter bool
}

// func NewSubscriber(id interface{}, msgChan chan Message, doFilter bool, msgTypes ...string) *Subscriber {
// 	return &Subscriber{id, msgChan, msgTypes, doFilter}
// }

// NewSubscriber return new Subscriber instance.
func NewSubscriber(id interface{}, msgChan chan Message, doFilter bool, msgType string, weight MessageWeight) *Subscriber {
	return &Subscriber{id, msgChan, msgType, weight, doFilter}
}

// ID return id.
func (s *Subscriber) ID() interface{} {
	return s.id
}

// MessageType return msgTypes.
func (s *Subscriber) MessageType() string {
	return s.msgType
}

// MessageChan return msgChan.
func (s *Subscriber) MessageChan() chan Message {
	return s.msgChan
}

// MessageWeight return weight of msgType
func (s *Subscriber) MessageWeight() MessageWeight {
	return s.msgWeight
}

// DoFilter return doFilter
func (s *Subscriber) DoFilter() bool {
	return s.doFilter
}
