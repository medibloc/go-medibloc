package dpos

import (
	"sort"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type State struct {
	candidateState *trie.Batch // key: addr, value: {addr, collateral, votePower}
	dynastyState   *trie.Batch // key: order, value: bpAddr

	stroage storage.Storage
}

func NewDposState(candidateStateHash []byte, dynastyStateHash []byte, stor storage.Storage) (core.DposState, error) {
	cs, err := trie.NewBatch(candidateStateHash, stor)
	if err != nil {
		return nil, err
	}
	ds, err := trie.NewBatch(dynastyStateHash, stor)
	if err != nil {
		return nil, err
	}

	return &State{
		candidateState: cs,
		dynastyState:   ds,
		stroage:        stor,
	}, nil
}

func (s *State) CandidateState() *trie.Batch {
	return s.candidateState
}

func (s *State) DynastyState() *trie.Batch {
	return s.dynastyState
}

func (s *State) Commit() error {
	if err := s.candidateState.Commit(); err != nil {
		return err
	}
	if err := s.dynastyState.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *State) RollBack() error {
	panic("implement me")
}

func (s *State) Clone() (core.DposState, error) {
	return NewDposState(s.candidateState.RootHash(), s.dynastyState.RootHash(), s.stroage)
}
func (s *State) BeginBatch() error {
	if err := s.candidateState.BeginBatch(); err != nil {
		return err
	}
	if err := s.dynastyState.BeginBatch(); err != nil {
		return err
	}
	return nil
}
func (s *State) RootBytes() ([]byte, error) {
	pbState := &dpospb.State{
		CandidateRootHash: s.CandidateState().RootHash(),
		DynastyRootHash:   s.DynastyState().RootHash(),
	}
	return proto.Marshal(pbState)
}

//SortByVotePower returns Descending ordered candidate slice
func SortByVotePower(cs *trie.Batch) ([]*dpospb.Candidate, error) {
	pbCandidates := make([]*dpospb.Candidate, 0)

	iter, err := cs.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, err := iter.Next()
	for exist {
		if err != nil {
			return nil, err
		}
		candidateBytes := iter.Value()

		pbCandidate := new(dpospb.Candidate)
		proto.Unmarshal(candidateBytes, pbCandidate)
		pbCandidates = append(pbCandidates, pbCandidate)

		exist, err = iter.Next()
	}

	var sortErr error
	sort.Slice(pbCandidates, func(i, j int) bool {
		x, err := util.NewUint128FromFixedSizeByteSlice(pbCandidates[i].VotePower)
		if err != nil {
			sortErr = err
			return false
		}
		y, err := util.NewUint128FromFixedSizeByteSlice(pbCandidates[j].VotePower)
		if err != nil {
			sortErr = err
			return false
		}
		// TODO @drSleepyTiger Secondary condition for same voting power
		return x.Cmp(y) > 0
	})

	if sortErr != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Sort Error")
		return nil, sortErr
	}

	return pbCandidates, nil
}

func MakeNewDynasty(sortedCandidates []*dpospb.Candidate, dynastySize int) []*common.Address {
	dynasty := make([]*common.Address, 0)

	for i := 0; i < dynastySize; i++ {
		addrBytes := sortedCandidates[i].Address
		addr := common.BytesToAddress(addrBytes)
		dynasty = append(dynasty, &addr)
	}
	// Todo @drsleepytiger ordering BP

	return dynasty
}

func findProposer(s core.DposState, timestamp int64) (common.Address, error) {
	ds := s.DynastyState()

	t := time.Duration(timestamp) * time.Second
	if t%BlockInterval != 0 {
		return common.Address{}, ErrInvalidBlockForgeTime
	}

	slotIndex := int32((t % DynastyInterval) / BlockInterval)
	addrBytes, err := ds.Get(byteutils.FromInt32(slotIndex))
	if err != nil {
		return common.Address{}, err
	}
	addr := common.BytesToAddress(addrBytes)
	return addr, nil
}

//SetDynastyState set dynasty state using dynasty slice
func SetDynastyState(ds *trie.Batch, dynasty []*common.Address) (error) {
	for i, addr := range dynasty {
		if err := ds.Put(byteutils.FromInt32(int32(i)), addr.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func dynastyStateToDynasty(dynastyState *trie.Batch) ([]*common.Address, error) {
	dynasty := make([]*common.Address, 0)
	iter, err := dynastyState.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, err := iter.Next()
	for exist {
		if err != nil {
			return nil, err
		}
		addr := common.BytesToAddress(iter.Value())
		dynasty = append(dynasty, &addr)

		exist, err = iter.Next()
	}
	return dynasty, nil
}

func checkTransitionDynasty(parentTimestamp int64, curTimestamp int64) bool {
	parentDynastyIndex := int(time.Duration(parentTimestamp) / DynastyInterval)
	curDynastyIndex := int(time.Duration(curTimestamp) / DynastyInterval)

	return curDynastyIndex > parentDynastyIndex
}

//MakeMintBlockDynasty returns dynasty slice for mint block
func (d *Dpos) MakeMintBlockDynasty(ts int64, parent *core.Block) ([]*common.Address, error) {
	if checkTransitionDynasty(parent.Timestamp(), ts) {
		cs := parent.State().DposState().CandidateState()
		sortedCandidates, err := SortByVotePower(cs)
		if err != nil {
			return nil, err
		}
		newDynasty := MakeNewDynasty(sortedCandidates, d.dynastySize)
		return newDynasty, nil
	}
	return dynastyStateToDynasty(parent.State().DposState().DynastyState())
}

func (d *Dpos) FindMintProposer(ts int64, parent *core.Block) (common.Address, error) {
	mintTs := nextMintSlot2(ts)
	dynasty, err := d.MakeMintBlockDynasty(mintTs, parent)
	if err != nil {
		return common.Address{}, err
	}
	proposerIndex := (time.Duration(ts) % DynastyInterval) / BlockInterval
	return *dynasty[proposerIndex], nil
}
