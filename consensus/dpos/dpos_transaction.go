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
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
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
	cs := b.State().DposState().CandidateState()
	as := b.State().AccState()

	_, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil && err != trie.ErrNotFound {
		return err
	}
	if err == nil {
		return ErrAlreadyCandidate
	}

	err = as.SubBalance(tx.candidateAddr.Bytes(), tx.collateral)
	if err != nil {
		return err
	}

	collateralBytes, err := tx.collateral.ToFixedSizeByteSlice()
	if err != nil {
		return err
	}

	zeroBytes, err := util.Uint128Zero().ToFixedSizeByteSlice()
	pbCandidate := &dpospb.Candidate{
		Address:   tx.candidateAddr.Bytes(),
		Collatral: collateralBytes,
		VotePower: zeroBytes,
	}
	candidateBytes, err := proto.Marshal(pbCandidate)
	if err != nil {
		return err
	}

	return cs.Put(tx.candidateAddr.Bytes(), candidateBytes)
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

	candidateBytes, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil {
		return ErrNotCandidate
	}

	// Refund collateral
	pbCandidate := new(dpospb.Candidate)
	err = proto.Unmarshal(candidateBytes, pbCandidate)
	if err != nil {
		return err
	}
	collateral, err := util.NewUint128FromFixedSizeByteSlice(pbCandidate.Collatral)
	if err != nil {
		return err
	}
	if err := as.AddBalance(tx.candidateAddr.Bytes(), collateral); err != nil {
		return err
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

	candidateBytes, err := cs.Get(tx.candidateAddr.Bytes())
	if err != nil {
		return ErrNotCandidate
	}

	voter, err := as.GetAccount(tx.voter.Bytes())
	if err != nil {
		return err
	}

	// Set voter's voted candidate
	if byteutils.Equal(voter.Voted(), tx.candidateAddr.Bytes()) {
		return ErrVoteDuplicate
	}
	as.SetVoted(tx.voter.Bytes(), tx.candidateAddr.Bytes())

	// Add voter's vesting to candidate's votePower
	pbCandidate := new(dpospb.Candidate)
	err = proto.Unmarshal(candidateBytes, pbCandidate)
	if err != nil {
		return err
	}
	votePower, err := util.NewUint128FromFixedSizeByteSlice(pbCandidate.VotePower)
	if err != nil {
		return err
	}
	newVotePower, err := votePower.Add(voter.Vesting())
	if err != nil {
		return err
	}
	pbCandidate.VotePower, err = newVotePower.ToFixedSizeByteSlice()
	if err != nil {
		return err
	}
	newBytesCandidate, err := proto.Marshal(pbCandidate)
	if err != nil {
		return err
	}

	return cs.Put(tx.candidateAddr.Bytes(), newBytesCandidate)
}
