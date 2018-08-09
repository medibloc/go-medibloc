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
	cs := b.State().DposState().CandidateState()
	as := b.State().AccState()

	_, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil && err != trie.ErrNotFound {
		return err
	}
	if err == nil {
		return ErrAlreadyCandidate
	}

	err = as.SubBalance(tx.candidateAddr, tx.collateral)
	if err != nil {
		return err
	}

	err = as.SetCollateral(tx.candidateAddr, tx.collateral)

	return cs.Put(tx.candidateAddr.Bytes(), tx.candidateAddr.Bytes())
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
	cs := b.State().DposState().CandidateState()
	as := b.State().AccState()

	_, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil {
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
		return cs.Delete(tx.candidateAddr.Bytes())
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

	return cs.Delete(tx.candidateAddr.Bytes())
}

//VoteTx is a structure for voting
type VoteTx struct {
	voter         common.Address
	candidateAddr common.Address
}

//NewVoteTx returns VoteTx
func NewVoteTx(tx *core.Transaction) (core.ExecutableTx, error) {
	return &VoteTx{
		voter:         tx.From(),
		candidateAddr: tx.To(),
	}, nil
}

//Execute VoteTx
func (tx *VoteTx) Execute(b *core.Block) error {
	cs := b.State().DposState().CandidateState()
	as := b.State().AccState()

	_, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil {
		return ErrNotCandidate
	}

	voter, err := as.GetAccount(tx.voter)
	if err != nil {
		return err
	}

	// Add voter's addr to candidate's voters
	if err := as.AddVoters(tx.candidateAddr, tx.voter); err != nil {
		return err
	}
	// Add voter's vesting to candidate's votePower
	if err := as.AddVotePower(tx.candidateAddr, voter.Vesting); err != nil {
		return err
	}
	// Add cadidate's addr to voter's voted
	if err := as.SetVoted(tx.voter, []common.Address{tx.candidateAddr}); err != nil {
		return err
	}

	return nil
}
