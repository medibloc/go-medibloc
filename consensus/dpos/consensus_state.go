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

package dpos

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// ConsensusState represents state for managing dynasty
type ConsensusState struct {
	dynasty     *trie.Trie
	dynastySize int
	proposer    common.Address
	timestamp   int64
	startTime   int64

	storage storage.Storage
}

// NewConsensusState returns new ConsensusState instance
func NewConsensusState(dynastyRootHash []byte, storage storage.Storage) (*ConsensusState, error) {
	t, err := trie.NewTrie(dynastyRootHash, storage)
	if err != nil {
		return nil, err
	}
	return &ConsensusState{
		dynasty: t,
		storage: storage,
	}, nil
}

// LoadConsensusState returns consensus state made from root bytes
func LoadConsensusState(rootBytes []byte, storage storage.Storage) (*ConsensusState, error) {
	cs, err := NewConsensusState(nil, storage)
	if err != nil {
		return nil, err
	}
	pb := new(consensuspb.ConsensusState)
	if err := proto.Unmarshal(rootBytes, pb); err != nil {
		return nil, err
	}
	if err := cs.FromProto(pb); err != nil {
		return nil, err
	}
	return cs, nil
}

// Timestamp returns timestamp
func (cs *ConsensusState) Timestamp() int64 {
	return cs.timestamp
}

// Proposer returns proposer
func (cs *ConsensusState) Proposer() common.Address {
	return cs.proposer
}

// InitDynasty sets all witnesses for the dynasty
func (cs *ConsensusState) InitDynasty(miners []*common.Address, dynastySize int, startTime int64) error {
	t, err := trie.NewTrie(nil, cs.storage)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create new trie.")
		return err
	}
	for _, addr := range miners {
		if err := t.Put(addr.Bytes(), addr.Bytes()); err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":     err,
				"address": addr.Hex(),
			}).Error("Failed to put dynasty members.")
			return err
		}
	}
	cs.dynasty = t
	cs.dynastySize = dynastySize
	cs.startTime = startTime
	cs.timestamp = startTime
	cs.proposer, err = FindProposer(startTime, miners)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"miners": miners,
			"time":   startTime,
		}).Error("Failed to find proposer.")
		return err
	}
	return nil
}

// Dynasty returns all witnesses in the dynasty
func (cs *ConsensusState) Dynasty() ([]*common.Address, error) {
	return TraverseDynasty(cs.dynasty)
}

// DynastySize returns dynasty size of dpos
func (cs *ConsensusState) DynastySize() int {
	return cs.dynastySize
}

// GetNextState returns consensus state at a certain time
func (cs *ConsensusState) GetNextState(at int64) (core.ConsensusState, error) {
	return cs.GetNextStateAfter(at - cs.timestamp)
}

// GetNextStateAfterGenesis returns consensus state after genesis block
func (cs *ConsensusState) GetNextStateAfterGenesis(timestamp int64) (core.ConsensusState, error) {
	deadline := mintDeadline(time.Unix(timestamp, 0))
	dynastyTrie, err := cs.dynasty.Clone()
	if err != nil {
		return nil, err
	}
	consensusState := &ConsensusState{
		dynasty:     dynastyTrie,
		dynastySize: cs.dynastySize,
		timestamp:   deadline.Unix(),
		startTime:   deadline.Unix(),
		storage:     cs.storage,
	}
	miners, err := TraverseDynasty(dynastyTrie)
	if err != nil {
		return nil, err
	}
	consensusState.proposer, err = FindProposer(deadline.Unix(), miners)
	if err != nil {
		return nil, err
	}
	return consensusState, nil
}

// GetNextStateAfter returns consensus state after certain amount of time
func (cs *ConsensusState) GetNextStateAfter(elapsedTime int64) (core.ConsensusState, error) {
	if cs.startTime+int64(DynastyInterval/time.Millisecond) < cs.timestamp+elapsedTime {
		return nil, core.ErrDynastyExpired
	}
	if elapsedTime < 0 || elapsedTime%int64(BlockInterval/time.Second) != 0 {
		return nil, ErrInvalidBlockForgeTime
	}
	dynastyTrie, err := cs.dynasty.Clone()
	if err != nil {
		return nil, err
	}
	consensusState := &ConsensusState{
		dynasty:     dynastyTrie,
		dynastySize: cs.dynastySize,
		timestamp:   cs.timestamp + elapsedTime,
		startTime:   cs.startTime,
		storage:     cs.storage,
	}
	miners, err := TraverseDynasty(dynastyTrie)
	if err != nil {
		return nil, err
	}
	consensusState.proposer, err = FindProposer(consensusState.timestamp, miners)
	if err != nil {
		return nil, err
	}
	return consensusState, nil
}

// ToProto returns protobuf version of consensus state
func (cs *ConsensusState) ToProto() proto.Message {
	return &consensuspb.ConsensusState{
		DynastyRoot: cs.dynasty.RootHash(),
		DynastySize: int64(cs.dynastySize),
		Proposer:    cs.proposer.Bytes(),
		StartTime:   cs.startTime,
		Timestamp:   cs.timestamp,
	}
}

// FromProto converts protobuf message to consensus state
func (cs *ConsensusState) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*consensuspb.ConsensusState); ok {
		t, err := trie.NewTrie(msg.DynastyRoot, cs.storage)
		if err != nil {
			return err
		}
		cs.dynasty = t
		cs.dynastySize = int(msg.DynastySize)
		cs.proposer = common.BytesToAddress(msg.Proposer)
		cs.timestamp = msg.Timestamp
		cs.startTime = msg.StartTime
		return nil
	}
	return ErrInvalidProtoToConsensusState
}

// RootBytes returns marshalled consensus state
func (cs *ConsensusState) RootBytes() ([]byte, error) {
	return proto.Marshal(cs.ToProto())
}

// Clone returns a clone of consensus state
func (cs *ConsensusState) Clone() (core.ConsensusState, error) {
	clone, err := NewConsensusState(nil, cs.storage)
	if err != nil {
		return nil, err
	}
	if err := clone.FromProto(cs.ToProto()); err != nil {
		return nil, err
	}
	return clone, nil
}

// FindProposer return proposer at the given time
func FindProposer(ts int64, miners []*common.Address) (common.Address, error) {
	now := time.Duration(ts) * time.Second
	if now%BlockInterval != 0 {
		return common.Address{}, ErrInvalidBlockForgeTime
	}
	offsetInDynastyInterval := now % DynastyInterval
	offsetInDynasty := int(offsetInDynastyInterval/BlockInterval) % len(miners)

	if int(offsetInDynasty) >= len(miners) {
		logging.WithFields(logrus.Fields{
			"offset": offsetInDynasty,
			"miners": len(miners),
		}).Error("No proposer selected for this turn.")
		return common.Address{}, ErrFoundNilProposer
	}
	return *(miners[offsetInDynasty]), nil
}

// TraverseDynasty traverses dynasty trie and return all miners found
func TraverseDynasty(dynasty *trie.Trie) (miners []*common.Address, err error) {
	members := []*common.Address{}
	iter, err := dynasty.Iterator(nil)
	if err != nil && err != storage.ErrKeyNotFound {
		return nil, err
	}
	if err != nil {
		return members, nil
	}
	exist, err := iter.Next()
	for exist {
		addr := common.BytesToAddress(iter.Value())
		members = append(members, &addr)
		exist, err = iter.Next()
	}
	return members, nil
}
