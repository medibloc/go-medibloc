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
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package dpos

import (
	"sort"
	"time"

	"bytes"

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

//State is a structure for dpos state
type State struct {
	candidateState *trie.Batch // key: addr, value: {addr, collateral, votePower}
	dynastyState   *trie.Batch // key: order, value: bpAddr

	stroage storage.Storage
}

//NewDposState returns new dpos state
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

//CandidateState returns candidate state
func (s *State) CandidateState() *trie.Batch {
	return s.candidateState
}

//DynastyState returns dynasty state
func (s *State) DynastyState() *trie.Batch {
	return s.dynastyState
}

//Commit saves batch to state
func (s *State) Commit() error {
	if err := s.candidateState.Commit(); err != nil {
		return err
	}
	if err := s.dynastyState.Commit(); err != nil {
		return err
	}
	return nil
}

//RollBack rollback batch
func (s *State) RollBack() error {
	if err := s.candidateState.RollBack(); err != nil {
		return err
	}
	if err := s.dynastyState.RollBack(); err != nil {
		return err
	}
	return nil
}

//Clone clone state
func (s *State) Clone() (core.DposState, error) {
	return NewDposState(s.candidateState.RootHash(), s.dynastyState.RootHash(), s.stroage)
}

//BeginBatch begin batch
func (s *State) BeginBatch() error {
	if err := s.candidateState.BeginBatch(); err != nil {
		return err
	}
	if err := s.dynastyState.BeginBatch(); err != nil {
		return err
	}
	return nil
}

//RootBytes returns root bytes for dpos state
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
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("failed to iterate")
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
			"err": sortErr,
		}).Error("Sort Error")
		return nil, sortErr
	}

	return pbCandidates, nil
}

//MakeNewDynasty returns new dynasty slice for new block
func MakeNewDynasty(sortedCandidates []*dpospb.Candidate, dynastySize int) []*common.Address {
	dynasty := make([]*common.Address, 0)

	for i := 0; i < dynastySize; i++ {
		addrBytes := sortedCandidates[i].Address
		addr := common.BytesToAddress(addrBytes)
		dynasty = append(dynasty, &addr)
	}
	// Todo @drsleepytiger ordering BP

	sort.Slice(dynasty, func(i, j int) bool {
		return bytes.Compare(dynasty[i].Bytes(), dynasty[j].Bytes()) < 0
	})

	return dynasty
}

//SetDynastyState set dynasty state using dynasty slice
func SetDynastyState(ds *trie.Batch, dynasty []*common.Address) error {
	for i, addr := range dynasty {
		if err := ds.Put(byteutils.FromInt32(int32(i)), addr.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

//DynastyStateToDynasty convert dynasty trie to address slice
func DynastyStateToDynasty(dynastyState *trie.Batch) ([]*common.Address, error) {
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

//CandidateStateToCandidates convert candidate trie to Candidate Proto
func CandidateStateToCandidates(candidateState *trie.Batch) ([]*dpospb.Candidate, error) {
	candidates := make([]*dpospb.Candidate, 0)

	iter, err := candidateState.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, err := iter.Next()
	for exist {
		if err != nil {
			return nil, err
		}
		pbCandidate := new(dpospb.Candidate)
		err = proto.Unmarshal(iter.Value(), pbCandidate)
		if err != nil {
			return nil, err
		}

		candidates = append(candidates, pbCandidate)
		exist, err = iter.Next()
	}
	return candidates, nil
}

func (d *Dpos) checkTransitionDynasty(parentTimestamp int64, curTimestamp int64) bool {
	parentDynastyIndex := int(time.Duration(parentTimestamp) * time.Second / d.DynastyInterval())
	curDynastyIndex := int(time.Duration(curTimestamp) * time.Second / d.DynastyInterval())

	return curDynastyIndex > parentDynastyIndex
}

//MakeMintDynasty returns dynasty slice for mint block
func (d *Dpos) MakeMintDynasty(ts int64, parent *core.Block) ([]*common.Address, error) {
	if d.checkTransitionDynasty(parent.Timestamp(), ts) {
		cs := parent.State().DposState().CandidateState()
		sortedCandidates, err := SortByVotePower(cs)
		if err != nil {
			return nil, err
		}
		newDynasty := MakeNewDynasty(sortedCandidates, d.dynastySize)
		return newDynasty, nil
	}
	return DynastyStateToDynasty(parent.State().DposState().DynastyState())
}

//FindMintProposer returns proposer for mint block
func (d *Dpos) FindMintProposer(ts int64, parent *core.Block) (common.Address, error) {
	mintTs := NextMintSlot2(ts)
	dynasty, err := d.MakeMintDynasty(mintTs, parent)
	if err != nil {
		return common.Address{}, err
	}
	proposerIndex := int(mintTs) % int(d.DynastyInterval().Seconds()) / int(BlockInterval.Seconds())
	return *dynasty[proposerIndex], nil
}

//SetMintDynastyState set mint block's dynasty state
func (d *Dpos) SetMintDynastyState(ts int64, parent *core.Block, block *core.Block) error {
	ds := block.State().DposState().DynastyState()

	dynasty, err := d.MakeMintDynasty(ts, parent)
	if err != nil {
		return err
	}

	err = SetDynastyState(ds, dynasty)
	if err != nil {
		return err
	}

	return nil
}

//Candidate returns candidate data in cadidateState
func (s *State) Candidate(addr common.Address) (*dpospb.Candidate, error) {
	cs := s.candidateState

	candidateBytes, err := cs.Get(addr.Bytes())
	if err != nil {
		return nil, ErrNotCandidate
	}

	pbCandidate := new(dpospb.Candidate)
	err = proto.Unmarshal(candidateBytes, pbCandidate)
	if err != nil {
		return nil, err
	}

	return pbCandidate, nil
}

// Candidates returns slice of candidate addresses
func (s *State) Candidates() ([]*dpospb.Candidate, error) {
	return CandidateStateToCandidates(s.candidateState)
}

// Dynasty returns slice of addresses in dynasty
func (s *State) Dynasty() ([]*common.Address, error) {
	return DynastyStateToDynasty(s.dynastyState)
}
