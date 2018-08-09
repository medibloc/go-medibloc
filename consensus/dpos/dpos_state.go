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
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

//State is a structure for dpos state
type State struct {
	candidateState *trie.Batch // key: addr, value: addr
	dynastyState   *trie.Batch // key: order, value: bpAddr

	storage storage.Storage
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
		storage:        stor,
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
	return NewDposState(s.candidateState.RootHash(), s.dynastyState.RootHash(), s.storage)
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
func SortByVotePower(as *core.AccountState, cs *trie.Batch) ([]common.Address, error) {
	//pbCandidates := make([]*dpospb.Candidate, 0)

	accounts := make([]*core.Account, 0)
	candidates := make([]common.Address, 0)

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
		addr := common.BytesToAddress(iter.Key())
		acc, err := as.GetAccount(addr)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)

		exist, err = iter.Next()
	}

	var sortErr error
	sort.Slice(accounts, func(i, j int) bool {
		// TODO @drSleepyTiger Secondary condition for same voting power
		return accounts[i].VotePower.Cmp(accounts[j].VotePower) > 0
	})

	if sortErr != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": sortErr,
		}).Error("Sort Error")
		return nil, sortErr
	}

	for _, v := range accounts {
		candidates = append(candidates, v.Address)
	}

	return candidates, nil
}

//MakeNewDynasty returns new dynasty slice for new block
func MakeNewDynasty(sortedCandidates []common.Address, dynastySize int) []common.Address {
	dynasty := make([]common.Address, 0)

	for i := 0; i < dynastySize; i++ {
		addr := sortedCandidates[i]
		dynasty = append(dynasty, addr)
	}
	// Todo @drsleepytiger ordering BP

	sort.Slice(dynasty, func(i, j int) bool {
		return bytes.Compare(dynasty[i].Bytes(), dynasty[j].Bytes()) < 0
	})

	return dynasty
}

//SetDynastyState set dynasty state using dynasty slice
func SetDynastyState(ds *trie.Batch, dynasty []common.Address) error {
	for i, addr := range dynasty {
		if err := ds.Put(byteutils.FromInt32(int32(i)), addr.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

//DynastyStateToDynasty convert dynasty trie to address slice
func DynastyStateToDynasty(dynastyState *trie.Batch) ([]common.Address, error) {
	dynasty := make([]common.Address, 0)
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
		dynasty = append(dynasty, addr)

		exist, err = iter.Next()
	}
	return dynasty, nil
}

//CandidateStateToCandidates convert candidate trie to Candidate Proto
func CandidateStateToCandidates(candidateState *trie.Batch) ([]common.Address, error) {
	candidates := make([]common.Address, 0)

	iter, err := candidateState.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, err := iter.Next()
	if err != nil {
		return nil, err
	}
	for exist {
		addr := common.BytesToAddress(iter.Key())
		candidates = append(candidates, addr)
		exist, err = iter.Next()
		if err != nil {
			return nil, err
		}
	}
	return candidates, nil
}

func (d *Dpos) checkTransitionDynasty(parentTimestamp int64, curTimestamp int64) bool {
	parentDynastyIndex := int(time.Duration(parentTimestamp) * time.Second / d.DynastyInterval())
	curDynastyIndex := int(time.Duration(curTimestamp) * time.Second / d.DynastyInterval())

	return curDynastyIndex > parentDynastyIndex
}

//MakeMintDynasty returns dynasty slice for mint block
func (d *Dpos) MakeMintDynasty(ts int64, parent *core.Block) ([]common.Address, error) {
	if d.checkTransitionDynasty(parent.Timestamp(), ts) {
		as := parent.State().AccState()
		cs := parent.State().DposState().CandidateState()
		sortedCandidates, err := SortByVotePower(as, cs)
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
	return dynasty[proposerIndex], nil
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

// IsCandidate returns true if addr is in candidateState
func (s *State) IsCandidate(addr common.Address) (bool, error) {
	cs := s.candidateState

	_, err := cs.Get(addr.Bytes())
	if err == trie.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// Candidates returns slice of addresses in candidates
func (s *State) Candidates() ([]common.Address, error) {
	return CandidateStateToCandidates(s.candidateState)
}

// Dynasty returns slice of addresses in dynasty
func (s *State) Dynasty() ([]common.Address, error) {
	return DynastyStateToDynasty(s.dynastyState)
}
