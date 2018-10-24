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
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/util"
)

// BecomeCandidateTx is a structure for quiting cadidate
type BecomeCandidateTx struct {
	candidateAddr common.Address
	collateral    *util.Uint128
}

//NewBecomeCandidateTx returns BecomeCandidateTx
func NewBecomeCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > core.MaxPayloadSize {
		return nil, core.ErrTooLargePayload
	}
	return &BecomeCandidateTx{
		candidateAddr: tx.From(),
		collateral:    tx.Value(),
	}, nil
}

//Execute NewBecomeCandidateTx
func (tx *BecomeCandidateTx) Execute(b *core.Block) error {
	// Add to candidate list
	ds := b.State().DposState()
	isCandidate, err := ds.IsCandidate(tx.candidateAddr)
	if err != nil {
		return err
	}
	if isCandidate {
		return ErrAlreadyCandidate
	}
	err = ds.PutCandidate(tx.candidateAddr)
	if err != nil {
		return err
	}
	// Set collateral to account's balance, and subtract from balance
	acc, err := b.State().GetAccount(tx.candidateAddr)
	if err != nil {
		return err
	}
	acc.Balance, err = acc.Balance.Sub(tx.collateral)
	if err == util.ErrUint128Underflow {
		return core.ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}
	acc.Collateral = tx.collateral

	return b.State().PutAccount(acc)
}

//Bandwidth returns bandwidth.
func (tx *BecomeCandidateTx) Bandwidth() (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	return core.TxBaseCPUBandwidth, core.TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}

//QuitCandidateTx is a structure for quiting candidate
type QuitCandidateTx struct {
	candidateAddr common.Address
}

//NewQuitCandidateTx returns QuitCandidateTx
func NewQuitCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > core.MaxPayloadSize {
		return nil, core.ErrTooLargePayload
	}
	return &QuitCandidateTx{
		candidateAddr: tx.From(),
	}, nil
}

//Execute QuitCandidateTx
func (tx *QuitCandidateTx) Execute(b *core.Block) error {
	ds := b.State().DposState()

	// Subtract from candidate list
	isCandidate, err := ds.IsCandidate(tx.candidateAddr)
	if err != nil {
		return err
	}
	if !isCandidate {
		return ErrNotCandidate
	}

	err = ds.DelCandidate(tx.candidateAddr)
	if err != nil {
		return err
	}

	// Refund collateral
	candidate, err := b.State().GetAccount(tx.candidateAddr)
	if err != nil {
		return err
	}

	candidate.Balance, err = candidate.Balance.Add(candidate.Collateral)
	if err != nil {
		return err
	}
	candidate.Collateral = util.NewUint128FromUint(0)
	candidate.VotePower, err = candidate.VotePower.Sub(candidate.VotePower)
	if err != nil {
		return err
	}
	oldVoters := candidate.VotersSlice()
	candidate.Voters, err = trie.NewBatch(nil, candidate.Storage)
	if err != nil {
		return err
	}
	err = b.State().PutAccount(candidate)
	if err != nil {
		return err
	}

	for _, v := range oldVoters {
		voter, err := b.State().GetAccount(common.BytesToAddress(v))
		if err != nil {
			return err
		}
		err = voter.Voted.Prepare()
		if err != nil {
			return err
		}
		err = voter.Voted.BeginBatch()
		if err != nil {
			return err
		}
		err = voter.Voted.Delete(tx.candidateAddr.Bytes())
		if err != nil {
			return err
		}
		err = voter.Voted.Commit()
		if err != nil {
			return err
		}
		err = voter.Voted.Flush()
		if err != nil {
			return nil
		}
		err = b.State().PutAccount(voter)
		if err != nil {
			return err
		}
	}
	return nil
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

//Bandwidth returns bandwidth.
func (tx *QuitCandidateTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return core.TxBaseCPUBandwidth, core.TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}

//VoteTx is a structure for voting
type VoteTx struct {
	voter      common.Address
	candidates []common.Address
}

//NewVoteTx returns VoteTx
func NewVoteTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > core.MaxPayloadSize {
		return nil, core.ErrTooLargePayload
	}
	payload := new(VotePayload)
	if err := core.BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}

	return &VoteTx{
		voter:      tx.From(),
		candidates: payload.Candidates,
	}, nil
}

//Execute VoteTx
func (tx *VoteTx) Execute(b *core.Block) error {
	ds := b.State().DposState()

	maxVote := b.Consensus().DynastySize() // TODO: max number of vote @drsleepytiger
	if len(tx.candidates) > maxVote {
		return ErrOverMaxVote
	}

	if checkDuplicate(tx.candidates) {
		return ErrDuplicateVote
	}

	acc, err := b.State().GetAccount(tx.voter)
	if err != nil {
		return err
	}
	myVesting := acc.Vesting
	oldVoted := acc.VotedSlice()

	acc.Voted, err = trie.NewBatch(nil, acc.Storage)
	if err != nil {
		return err
	}
	err = acc.Voted.Prepare()
	if err != nil {
		return err
	}
	err = acc.Voted.BeginBatch()
	if err != nil {
		return err
	}
	for _, c := range tx.candidates {
		// Add candidate to voters voted
		err = acc.Voted.Put(c.Bytes(), c.Bytes())
		if err != nil {
			return err
		}
	}
	err = acc.Voted.Commit()
	if err != nil {
		return err
	}
	err = acc.Voted.Flush()
	if err != nil {
		return err
	}
	err = b.State().PutAccount(acc)
	if err != nil {
		return err
	}

	for _, addrBytes := range oldVoted {
		candidate, err := b.State().GetAccount(common.BytesToAddress(addrBytes))
		if err != nil {
			return err
		}
		candidate.VotePower, err = candidate.VotePower.Sub(myVesting)
		if err != nil {
			return err
		}
		err = candidate.Voters.Prepare()
		if err != nil {
			return err
		}
		err = candidate.Voters.BeginBatch()
		if err != nil {
			return err
		}
		err = candidate.Voters.Delete(tx.voter.Bytes())
		if err != nil {
			return err
		}
		err = candidate.Voters.Commit()
		if err != nil {
			return err
		}
		err = candidate.Voters.Flush()
		if err != nil {
			return err
		}
		err = b.State().PutAccount(candidate)
		if err != nil {
			return err
		}
	}

	for _, c := range tx.candidates {
		isCandidate, err := ds.IsCandidate(c)
		if err != nil {
			return err
		}
		if !isCandidate {
			return ErrNotCandidate
		}

		candidate, err := b.State().GetAccount(c)
		if err != nil {
			return err
		}
		candidate.VotePower, err = candidate.VotePower.Add(myVesting)
		if err != nil {
			return err
		}
		err = candidate.Voters.Prepare()
		if err != nil {
			return err
		}
		err = candidate.Voters.BeginBatch()
		if err != nil {
			return err
		}
		err = candidate.Voters.Put(tx.voter.Bytes(), tx.voter.Bytes())
		if err != nil {
			return err
		}
		err = candidate.Voters.Commit()
		if err != nil {
			return err
		}
		err = candidate.Voters.Flush()
		if err != nil {
			return err
		}
		err = b.State().PutAccount(candidate)
		if err != nil {
			return err
		}
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

//Bandwidth returns bandwidth.
func (tx *VoteTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return core.TxBaseCPUBandwidth, core.TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}
