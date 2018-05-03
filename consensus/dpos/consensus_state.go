package dpos

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// ConsensusState represents state for managing dynasty
type ConsensusState struct {
	dynasty   *trie.Trie
	proposer  common.Address
	timestamp int64
	startTime int64

	storage storage.Storage
}

// NewConsensusState returns new ConsensusState instance
func NewConsensusState(rootHash []byte, storage storage.Storage) (*ConsensusState, error) {
	t, err := trie.NewTrie(rootHash, storage)
	if err != nil {
		return nil, err
	}
	return &ConsensusState{
		dynasty: t,
		storage: storage,
	}, nil
}

// InitDynasty sets all witnesses for the dynasty
func (cs *ConsensusState) InitDynasty(miners []*common.Address, startTime int64) error {
	t, err := trie.NewTrie(nil, cs.storage)
	if err != nil {
		return err
	}
	for _, addr := range miners {
		if err := t.Put(addr.Bytes(), addr.Bytes()); err != nil {
			return err
		}
	}
	cs.dynasty = t
	cs.startTime = startTime
	cs.timestamp = int64(0)
	cs.proposer = common.Address{}
	return nil
}

// Dynasty returns all witnesses in the dynasty
func (cs *ConsensusState) Dynasty() ([]*common.Address, error) {
	return TraverseDynasty(cs.dynasty)
}

// GetNextState returns consensus state at a certain time
func (cs *ConsensusState) GetNextState(at int64) (*ConsensusState, error) {
	return cs.GetNextStateAfter(at - cs.timestamp)
}

// GetNextStateAfter returns consensus state after certain amount of time
func (cs *ConsensusState) GetNextStateAfter(elapsedTime int64) (*ConsensusState, error) {
	if cs.startTime+int64(DynastyInterval/time.Millisecond) < cs.timestamp+elapsedTime {
		return nil, ErrDynastyExpired
	}
	if elapsedTime < 0 || elapsedTime%int64(BlockInterval/time.Millisecond) != 0 {
		return nil, ErrInvalidBlockForgeTime
	}
	dynastyTrie, err := cs.dynasty.Clone()
	if err != nil {
		return nil, err
	}
	consensusState := &ConsensusState{
		dynasty:   dynastyTrie,
		timestamp: cs.timestamp + elapsedTime,
		storage:   cs.storage,
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

// ConsensusRoot returns root bytes of consensus state
func (cs *ConsensusState) ConsensusRoot() ([]byte, error) {
	pbCs := &consensuspb.ConsensusState{
		DynastyRoot: cs.dynasty.RootHash(),
		Proposer:    cs.proposer.Bytes(),
		Timestamp:   cs.timestamp,
	}
	csBytes, err := proto.Marshal(pbCs)
	if err != nil {
		return nil, err
	}
	return csBytes, nil
}

// FromProto converts proto buf messgae to consensus state
func (cs *ConsensusState) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*consensuspb.ConsensusState); ok {
		t, err := trie.NewTrie(msg.DynastyRoot, cs.storage)
		if err != nil {
			return err
		}
		cs.dynasty = t
		cs.proposer = common.BytesToAddress(msg.Proposer)
		cs.timestamp = msg.Timestamp
		return nil
	}
	return ErrInvalidProtoToConsensusState
}

// FindProposer return proposer at the given time
func FindProposer(ts int64, miners []*common.Address) (common.Address, error) {
	now := time.Duration(ts) * time.Second
	if now%BlockInterval != 0 {
		return common.Address{}, ErrInvalidBlockForgeTime
	}
	offsetInDynastyInterval := now % DynastyInterval
	offsetInDynasty := offsetInDynastyInterval % DynastySize

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
