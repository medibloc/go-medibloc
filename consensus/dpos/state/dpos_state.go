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

package dposstate

import (
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	dpospb "github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// TODO applying prefix
// Prefixes for blockState trie
const (
	CandidatePrefix = "c_" // alias account prefix for account state trie
	DynastyPrefix   = "d_" // alias account prefix for account state trie
)

// State is a structure for dpos state
type State struct {
	candidateState *trie.Batch // key: candidate id(txHash), value: candidate
	dynastyState   *trie.Batch // key: order, value: bpAddr
}

// NewDposState returns new dpos state
func NewDposState(candidateStateHash []byte, dynastyStateHash []byte, stor storage.Storage) (*State, error) {
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
	}, nil
}

// GetCandidate returns candidate from candidate state
func (s *State) GetCandidate(cid []byte) (*Candidate, error) {
	candidate := new(Candidate)
	if err := s.candidateState.GetData(append([]byte(CandidatePrefix), cid...), candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

// PutCandidate puts candidate to candidate state
func (s *State) PutCandidate(cid []byte, candidate *Candidate) error {
	return s.candidateState.PutData(append([]byte(CandidatePrefix), cid...), candidate)
}

// DelCandidate del candidate from candidate state
func (s *State) DelCandidate(cid []byte) error {
	return s.candidateState.Delete(append([]byte(CandidatePrefix), cid...))
}

// GetCandidates returns candidate list from candidate state.
func (s *State) GetCandidates() ([]*Candidate, error) {
	candidates := make([]*Candidate, 0)

	iter, err := s.candidateState.Iterator([]byte(CandidatePrefix))
	if err != nil {
		return nil, err
	}
	for {
		exist, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !exist {
			break
		}
		candidate := new(Candidate)
		err = candidate.FromBytes(iter.Value())
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// GetProposer returns proposer address of index
func (s *State) GetProposer(index int) (common.Address, error) {
	ds := s.dynastyState
	b, err := ds.Get(append([]byte(DynastyPrefix), byteutils.FromInt32(int32(index))...))
	if err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(b)
}

// PutProposer sets proposer address to index
func (s *State) PutProposer(index int, addr common.Address) error {
	ds := s.dynastyState
	return ds.Put(append([]byte(DynastyPrefix), byteutils.FromInt32(int32(index))...), addr.Bytes())
}

func (s *State) applyBatchToEachState(fn trie.BatchCallType) error {
	err := fn(s.candidateState)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to apply batch function to candidate state.")
		return err
	}
	err = fn(s.dynastyState)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to apply batch function to dynasty state.")
		return err
	}
	return nil
}

// Prepare prepare trie
func (s *State) Prepare() error {
	err := s.applyBatchToEachState(trie.PrepareCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to prepare dpos state.")
		return err
	}
	return nil
}

// BeginBatch begin batch
func (s *State) BeginBatch() error {
	err := s.applyBatchToEachState(trie.BeginBatchCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to begin batch dpos state.")
		return err
	}
	return nil
}

// Commit saves batch to state
func (s *State) Commit() error {
	err := s.applyBatchToEachState(trie.CommitCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to commit dpos state.")
		return err
	}
	return nil
}

// RollBack rollback batch
func (s *State) RollBack() error {
	err := s.applyBatchToEachState(trie.RollbackCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to rollback dpos state.")
		return err
	}
	return nil
}

// Flush flush data to storage
func (s *State) Flush() error {
	err := s.applyBatchToEachState(trie.FlushCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to flush dpos state.")
		return err
	}
	return nil
}

// Reset reset trie's refCounter
func (s *State) Reset() error {
	err := s.applyBatchToEachState(trie.ResetCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to reset dpos state.")
		return err
	}
	return nil
}

// Clone clone state
func (s *State) Clone() (*State, error) {
	cs, err := s.candidateState.Clone()
	if err != nil {
		return nil, err
	}
	ds, err := s.dynastyState.Clone()
	if err != nil {
		return nil, err
	}

	return &State{
		candidateState: cs,
		dynastyState:   ds,
	}, nil
}

// RootBytes returns root bytes for dpos state
func (s *State) RootBytes() ([]byte, error) {
	csRoot, err := s.candidateState.RootHash()
	if err != nil {
		return nil, err
	}
	dsRoot, err := s.dynastyState.RootHash()
	if err != nil {
		return nil, err
	}

	pbState := &dpospb.State{
		CandidateRootHash: csRoot,
		DynastyRootHash:   dsRoot,
	}
	return proto.Marshal(pbState)
}

// SortByVotePower returns Descending ordered candidate slice
func (s *State) SortByVotePower() ([]common.Address, error) {
	addresses := make([]common.Address, 0)

	candidates, err := s.GetCandidates()
	if err != nil {
		return nil, err
	}

	var sortErr error
	sort.Slice(candidates, func(i, j int) bool {
		// TODO @drSleepyTiger Secondary condition for same voting power
		return candidates[i].VotePower.Cmp(candidates[j].VotePower) > 0
	})

	if sortErr != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": sortErr,
		}).Error("Sort Error")
		return nil, sortErr
	}

	for _, v := range candidates {
		addresses = append(addresses, v.Addr)
	}

	return addresses, nil
}

// SetDynasty set dynastyState by using dynasty slice
func (s *State) SetDynasty(dynasty []common.Address) error {
	for i, addr := range dynasty {
		if err := s.PutProposer(i, addr); err != nil {
			return err
		}
	}
	return nil
}

// AddVotePowerToCandidate add vote power to candidate
func (s *State) AddVotePowerToCandidate(id []byte, amount *util.Uint128) error {
	candidate := new(Candidate)
	err := s.candidateState.GetData(append([]byte(CandidatePrefix), id...), candidate)
	if err != nil {
		return err
	}

	candidate.VotePower, err = candidate.VotePower.Add(amount)
	if err != nil {
		return err
	}
	return s.candidateState.PutData(append([]byte(CandidatePrefix), id...), candidate)
}

// SubVotePowerToCandidate sub vote power from candidate's vote power
func (s *State) SubVotePowerToCandidate(id []byte, amount *util.Uint128) error {
	candidate := new(Candidate)
	err := s.candidateState.GetData(append([]byte(CandidatePrefix), id...), candidate)
	if err != nil {
		return err
	}

	candidate.VotePower, err = candidate.VotePower.Sub(amount)
	if err != nil {
		return err
	}
	return s.candidateState.PutData(append([]byte(CandidatePrefix), id...), candidate)
}
