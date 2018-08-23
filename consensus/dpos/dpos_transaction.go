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
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/t/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// BecomeCandidateTx is a structure for quiting cadidate
type BecomeCandidateTx struct {
	candidateAddr common.Address
	collateral    *util.Uint128
}

//NewBecomeCandidateTx returns BecomeCandidateTx
func NewBecomeCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	return &BecomeCandidateTx{
		candidateAddr: tx.From(),
		collateral:    tx.Value(),
	}, nil
}

//Execute NewBecomeCandidateTx
func (tx *BecomeCandidateTx) Execute(b *core.Block) error {
	as := b.State().AccState()
	ds := b.State().DposState()

	isCandidate, err := ds.IsCandidate(tx.candidateAddr)
	if err != nil {
		return err
	}
	if isCandidate {
		return ErrAlreadyCandidate
	}

	err = as.SubBalance(tx.candidateAddr, tx.collateral)
	if err != nil {
		return err
	}

	err = as.SetCollateral(tx.candidateAddr, tx.collateral)

	return ds.PutCandidate(tx.candidateAddr)
}

//QuitCandidateTx is a structure for quiting cadidate
type QuitCandidateTx struct {
	candidateAddr common.Address
}

//NewQuitCandidateTx returns QuitCandidateTx
func NewQuitCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	return &QuitCandidateTx{
		candidateAddr: tx.From(),
	}, nil
}

//Execute QuitCandidateTx
func (tx *QuitCandidateTx) Execute(b *core.Block) error {
	ds := b.State().DposState()
	as := b.State().AccState()

	isCandidate, err := ds.IsCandidate(tx.candidateAddr)
	if err != nil {
		return err
	}
	if !isCandidate {
		return ErrNotCandidate
	}

	// Refund collateral
	acc, err := as.GetAccount(tx.candidateAddr)
	if err != nil {
		return err
	}

	if err := as.AddBalance(tx.candidateAddr, acc.Collateral); err != nil {
		return err
	}

	if err := as.SetCollateral(tx.candidateAddr, util.NewUint128FromUint(0)); err != nil {
		return err
	}

	if err := as.SubVotePower(tx.candidateAddr, acc.VotePower); err != nil {
		return err
	}

	voters := acc.Voters
	if voters.RootHash() == nil {
		return ds.DelCandidate(tx.candidateAddr)
	}

	iter, err := voters.Iterator(nil)
	if err != nil {
		return err
	}
	exist, err := iter.Next()
	if err != nil {
		return err
	}
	for exist {
		voterAddr := common.BytesToAddress(iter.Key())
		err = as.SubVoted(voterAddr, tx.candidateAddr)
		if err != nil {
			return err
		}
		err = as.SubVoters(tx.candidateAddr, voterAddr)
		if err != nil {
			return err
		}

		exist, err = iter.Next()
		if err != nil {
			return err
		}
	}

	return ds.DelCandidate(tx.candidateAddr)
}

// VotePayload is payload type for VoteTx
type VotePayload struct {
	Candidates []common.Address
}

// FromBytes converts bytes to payload.
func (payload *VotePayload) FromBytes(b []byte) error {
	payloadPb := &corepb.VotePayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	var candidates []common.Address
	for _, candidate := range payloadPb.Candidates {
		candidates = append(candidates, common.BytesToAddress(candidate))
	}
	payload.Candidates = candidates
	return nil
}

// ToBytes returns marshaled RevokeCertificationPayload
func (payload *VotePayload) ToBytes() ([]byte, error) {
	candidates := make([][]byte, 0)
	for _, candidate := range payload.Candidates {
		candidates = append(candidates, candidate.Bytes())
	}

	payloadPb := &corepb.VotePayload{
		Candidates: candidates,
	}
	return proto.Marshal(payloadPb)
}

//VoteTx is a structure for voting
type VoteTx struct {
	voter      common.Address
	candidates []common.Address
}

//NewVoteTx returns VoteTx
func NewVoteTx(tx *core.Transaction) (core.ExecutableTx, error) {
	payload := new(VotePayload)
	if err := payload.FromBytes(tx.Payload()); err != nil {
		return nil, err
	}
	bytes, err := payload.ToBytes()
	if err != nil {
		return nil, core.ErrInvalidTxPayload
	}
	if byteutils.Bytes2Hex(bytes) != byteutils.Bytes2Hex(tx.Payload()) {
		return nil, core.ErrInvalidTxPayload
	}

	return &VoteTx{
		voter:      tx.From(),
		candidates: payload.Candidates,
	}, nil
}

//Execute VoteTx
func (tx *VoteTx) Execute(b *core.Block) error {
	ds := b.State().DposState()
	as := b.State().AccState()

	maxVote := b.Consensus().DynastySize() // TODO: max number of vote @drsleepytiger
	if len(tx.candidates) > maxVote {
		return ErrOverMaxVote
	}

	if checkDuplicate(tx.candidates) {
		return ErrDuplicateVote
	}

	acc, err := as.GetAccount(tx.voter)
	if err != nil {
		return err
	}

	myVesting := acc.Vesting
	oldVoted := core.KeyTrieToSlice(acc.Voted)

	for _, addrBytes := range oldVoted {
		candidate := common.BytesToAddress(addrBytes)
		as.SubVoters(candidate, tx.voter)
		as.SubVotePower(candidate, myVesting)
	}
	acc.Voted.SetRootHash(nil)

	for _, addr := range tx.candidates {
		isCandidate, err := ds.IsCandidate(addr)
		if err != nil {
			return err
		}
		if !isCandidate {
			return ErrNotCandidate
		}

		voter, err := as.GetAccount(tx.voter)
		if err != nil {
			return err
		}

		// Add voter's addr to candidate's voters
		if err := as.AddVoters(addr, tx.voter); err != nil {
			return err
		}
		// Add voter's vesting to candidate's votePower
		if err := as.AddVotePower(addr, voter.Vesting); err != nil {
			return err
		}
	}
	// Add cadidate's addr to voter's voted
	if err := as.SetVoted(tx.voter, tx.candidates); err != nil {
		return err
	}

	return nil
}

func checkDuplicate(candidates []common.Address) bool {
	temp := make(map[common.Address]bool, 0)
	for _, v := range candidates {
		temp[v] = true
	}
	if len(temp) != len(candidates) {
		return true
	}
	return false
}
