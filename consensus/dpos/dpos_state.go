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
	"bytes"
	"sort"

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
func (s *State) SortByVotePower(as *core.AccountState) ([]common.Address, error) {
	//pbCandidates := make([]*dpospb.Candidate, 0)

	cs := s.candidateState

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

func (d *Dpos) checkTransitionDynasty(parentTimestamp int64, curTimestamp int64) bool {
	parentDynastyIndex := d.calcDynastyIndex(parentTimestamp)
	curDynastyIndex := d.calcDynastyIndex(curTimestamp)

	return curDynastyIndex > parentDynastyIndex
}

//MakeMintDynasty returns dynasty slice for mint block
func (d *Dpos) MakeMintDynasty(ts int64, parent *core.Block) ([]common.Address, error) {
	if d.checkTransitionDynasty(parent.Timestamp(), ts) {
		as := parent.State().AccState()
		ds := parent.State().DposState()
		sortedCandidates, err := ds.SortByVotePower(as)
		if err != nil {
			return nil, err
		}
		newDynasty := MakeNewDynasty(sortedCandidates, d.dynastySize)
		return newDynasty, nil
	}
	return parent.State().DposState().Dynasty()
}

//FindMintProposer returns proposer for mint block
func (d *Dpos) FindMintProposer(ts int64, parent *core.Block) (common.Address, error) {
	mintTs := NextMintSlot2(ts)
	dynasty, err := d.MakeMintDynasty(mintTs, parent)
	if err != nil {
		return common.Address{}, err
	}
	proposerIndex := d.calcProposerIndex(mintTs)
	return dynasty[proposerIndex], nil
}

//SetDynasty set dynastyState by using dynasty slice
func (s *State) SetDynasty(dynasty []common.Address) error {
	for i, addr := range dynasty {
		if err := s.dynastyState.Put(byteutils.FromInt32(int32(i)), addr.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

//SetMintDynastyState set mint block's dynasty state
func (s *State) SetMintDynastyState(ts int64, parent *core.Block, dynastySize int) error {
	d := New(dynastySize)

	mintDynasty, err := d.MakeMintDynasty(ts, parent)
	if err != nil {
		return err
	}
	s.SetDynasty(mintDynasty)
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

//InDynasty returns true if addr is in dynastyState
func (s *State) InDynasty(addr common.Address) (bool, error) {
	ds := s.dynastyState
	addrBytes := addr.Bytes()

	iter, err := ds.Iterator(nil)
	if err != nil {
		return false, err
	}

	exist, err := iter.Next()
	if err != nil {
		return false, err
	}
	for exist {
		if byteutils.Equal(iter.Value(), addrBytes) {
			return true, nil
		}
		exist, err = iter.Next()
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

// Candidates returns slice of addresses in candidates
func (s *State) Candidates() ([]common.Address, error) {
	cs := s.candidateState

	candidates := make([]common.Address, 0)
	iter, err := cs.Iterator(nil)
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

// Dynasty returns slice of addresses in dynasty
func (s *State) Dynasty() ([]common.Address, error) {
	ds := s.dynastyState

	dynasty := make([]common.Address, 0)
	iter, err := ds.Iterator(nil)
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

//PutCandidate put candidate to candidate state
func (s *State) PutCandidate(addr common.Address) error {
	return s.candidateState.Put(addr.Bytes(), addr.Bytes())
}

//DelCandidate delete candidate from candidate state
func (s *State) DelCandidate(addr common.Address) error {
	return s.candidateState.Delete(addr.Bytes())
}
