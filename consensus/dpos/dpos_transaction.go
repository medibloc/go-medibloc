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
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// BecomeCandidateTx is a structure for quiting cadidate
type BecomeCandidateTx struct {
	txHash        []byte
	candidateAddr common.Address
	collateral    *util.Uint128
	payload       *BecomeCandidatePayload
	size          int
}

//NewBecomeCandidateTx returns BecomeCandidateTx
func NewBecomeCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > core.MaxPayloadSize {
		return nil, core.ErrTooLargePayload
	}
	payload := new(BecomeCandidatePayload)
	if err := core.BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, core.ErrInvalidAddress
	}
	if !common.IsHash(byteutils.Bytes2Hex(tx.Hash())) {
		return nil, core.ErrInvalidHash
	}

	return &BecomeCandidateTx{
		txHash:        tx.Hash(),
		candidateAddr: tx.From(),
		collateral:    tx.Value(),
		payload:       payload,
		size:          size,
	}, nil
}

//Execute NewBecomeCandidateTx
func (tx *BecomeCandidateTx) Execute(b *core.Block) error {
	acc, err := b.State().GetAccount(tx.candidateAddr)
	if err != nil {
		return err
	}

	if acc.CandidateID != nil {
		return ErrAlreadyCandidate
	}
	_, err = acc.GetData(core.AliasPrefix, []byte(common.AliasKey))
	if err == core.ErrNotFound {
		return core.ErrAliasNotExist
	}

	minimumCollateral, err := util.NewUint128FromString(MinimumCandidateCollateral)
	if err != nil {
		return err
	}
	if tx.collateral.Cmp(minimumCollateral) < 0 {
		return ErrNotEnoughCandidateCollateral
	}

	// Subtract collateral from balance
	acc.Balance, err = acc.Balance.Sub(tx.collateral)
	if err == util.ErrUint128Underflow {
		return core.ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}

	acc.CandidateID = tx.txHash

	if err := b.State().PutAccount(acc); err != nil {
		return nil
	}

	//TODO: URL 유효성 확인? Regex? @shwankim

	candidate := &Candidate{
		ID:         tx.txHash,
		Addr:       tx.candidateAddr,
		Collateral: tx.collateral,
		VotePower:  util.NewUint128(),
		URL:        tx.payload.URL,
		Timestamp:  b.Timestamp(),
	}

	// Add candidate to candidate state
	cs := b.State().DposState().CandidateState()

	return cs.PutData(tx.txHash, candidate)
}

//Bandwidth returns bandwidth.
func (tx *BecomeCandidateTx) Bandwidth() (cpuUsage uint64, netUsage uint64) {
	return 1000, uint64(tx.size)
}

//QuitCandidateTx is a structure for quiting candidate
type QuitCandidateTx struct {
	candidateAddr common.Address
	size          int
}

//NewQuitCandidateTx returns QuitCandidateTx
func NewQuitCandidateTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > core.MaxPayloadSize {
		return nil, core.ErrTooLargePayload
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, core.ErrInvalidAddress
	}

	return &QuitCandidateTx{
		candidateAddr: tx.From(),
		size:          size,
	}, nil
}

//Execute QuitCandidateTx
func (tx *QuitCandidateTx) Execute(b *core.Block) error {
	acc, err := b.State().GetAccount(tx.candidateAddr)
	if err != nil {
		return err
	}
	if acc.CandidateID == nil {
		return ErrNotCandidate
	}

	cs := b.State().DposState().CandidateState()
	candidate := new(Candidate)
	err = cs.GetData(acc.CandidateID, candidate)
	if err == trie.ErrNotFound {
		return ErrNotCandidate
	} else if err != nil {
		return err
	}

	cID := acc.CandidateID
	// Remove candidateID on account
	acc.CandidateID = nil

	// Refund collateral to account
	acc.Balance, err = acc.Balance.Add(candidate.Collateral)
	if err != nil {
		return err
	}
	// save changed account
	if err := b.State().PutAccount(acc); err != nil {
		return nil
	}

	// change quit flag on candidate
	if err := cs.Delete(cID); err != nil {
		return err
	}

	return nil
}

//Bandwidth returns bandwidth.
func (tx *QuitCandidateTx) Bandwidth() (cpuUsage uint64, netUsage uint64) {
	return 1000, uint64(tx.size)
}

//VoteTx is a structure for voting
type VoteTx struct {
	voter        common.Address
	candidateIDs [][]byte
	size         int
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
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, core.ErrInvalidAddress
	}

	return &VoteTx{
		voter:        tx.From(),
		candidateIDs: payload.CandidateIDs,
		size:         size,
	}, nil
}

//Execute VoteTx
func (tx *VoteTx) Execute(b *core.Block) error {
	if len(tx.candidateIDs) > MaxVote {
		return ErrOverMaxVote
	}

	acc, err := b.State().GetAccount(tx.voter)
	if err != nil {
		return err
	}

	newVoted, err := trie.NewBatch(nil, acc.Storage)
	if err != nil {
		return err
	}
	err = newVoted.Prepare()
	if err != nil {
		return err
	}
	err = newVoted.BeginBatch()
	if err != nil {
		return err
	}
	for _, c := range tx.candidateIDs {
		_, err := newVoted.Get(c)
		// Check duplicated vote
		if err != nil && err != trie.ErrNotFound {
			return err
		} else if err == nil {
			return ErrDuplicateVote
		}

		// Add candidate to voters voted
		if err := newVoted.Put(c, c); err != nil {
			return err
		}
	}
	err = newVoted.Commit()
	if err != nil {
		return err
	}
	err = newVoted.Flush()
	if err != nil {
		return err
	}

	ds := b.State().DposState()

	// Add vote power to new voted
	iter, err := newVoted.Iterator(nil)
	if err != nil {
		return err
	}
	for {
		exist, err := iter.Next()
		if err != nil {
			return err
		}
		if !exist {
			break
		}

		candidateID := iter.Key()
		err = ds.AddVotePowerToCandidate(candidateID, acc.Staking)
		if err == core.ErrCandidateNotFound {
			return ErrNotCandidate
		} else if err != nil {
			return err
		}
	}

	oldVoted := acc.Voted
	// Subtract vote power from old voted
	iter, err = oldVoted.Iterator(nil)
	if err != nil {
		return err
	}
	for {
		exist, err := iter.Next()
		if err != nil {
			return err
		}
		if !exist {
			break
		}

		candidateID := iter.Key()
		err = ds.SubVotePowerToCandidate(candidateID, acc.Staking)
		if err == core.ErrCandidateNotFound {
			continue // candidate quited
		} else if err != nil {
			return err
		}
	}

	// change voter's account
	acc.Voted = newVoted
	if err := b.State().PutAccount(acc); err != nil {
		return err
	}

	return nil
}

//Bandwidth returns bandwidth.
func (tx *VoteTx) Bandwidth() (cpuUsage uint64, netUsage uint64) {
	return 1000, uint64(tx.size)
}
