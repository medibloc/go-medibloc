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
	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/medibloc/go-medibloc/crypto/hash"
	netpb "github.com/medibloc/go-medibloc/net/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// RecvMessage is a struct for received message
type RecvMessage struct {
	pb     *netpb.Message
	sender peer.ID
	hash   string
}

// MessageType returns message type
func (m *RecvMessage) MessageType() string {
	return m.pb.Type
}

// MessageFrom returns peer id of message sender
func (m *RecvMessage) MessageFrom() string {
	return m.sender.Pretty()
}

// Data returns data
func (m *RecvMessage) Data() []byte {
	return m.pb.Data
}

// Hash returns hash
func (m *RecvMessage) Hash() string {
	return m.hash
}

func newRecvMessage(bytes []byte, sender peer.ID) (*RecvMessage, error) {
	pb := new(netpb.Message)
	if err := proto.Unmarshal(bytes, pb); err != nil {
		return nil, err
	}
	return &RecvMessage{
		pb:     pb,
		sender: sender,
		hash:   byteutils.Bytes2Hex(hash.Sha3256(bytes)),
	}, nil
}

// SendMessage is a struct for sending message
type SendMessage struct {
	bytes    []byte
	receiver peer.ID
	priority int
	hash     string
}

// SetReceiver sets receiver
func (m *SendMessage) SetReceiver(receiver peer.ID) {
	m.receiver = receiver
}

// Bytes returns bytes
func (m *SendMessage) Bytes() []byte {
	return m.bytes
}

// MessageType returns message type
func (m *SendMessage) MessageType() string {
	pb := new(netpb.Message)
	if err := proto.Unmarshal(m.bytes, pb); err != nil {
		return ""
	}
	return pb.Type
}

// Data returns data
func (m *SendMessage) Data() []byte {
	pb := new(netpb.Message)
	if err := proto.Unmarshal(m.bytes, pb); err != nil {
		return []byte{}
	}
	return pb.Data
}

// Hash returns hash
func (m *SendMessage) Hash() string {
	return m.hash
}

func newSendMessage(chainID uint32, msgType string, data []byte, priority int, receiver peer.ID) (*SendMessage, error) {
	pb := &netpb.Message{
		ChainId: chainID,
		Type:    msgType,
		Data:    data,
	}

	bytes, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}

	return &SendMessage{
		bytes:    bytes,
		receiver: receiver,
		priority: priority,
		hash:     byteutils.Bytes2Hex(hash.Sha3256(bytes)),
	}, nil
}

func (m *SendMessage) copy() *SendMessage {
	return &SendMessage{
		bytes:    m.bytes,
		receiver: m.receiver,
		priority: m.priority,
		hash:     m.hash,
	}
}
