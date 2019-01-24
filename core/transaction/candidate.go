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

package transaction

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	dState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	"github.com/medibloc/go-medibloc/core"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util"
)

// BecomeCandidatePayload is payload type for BecomeCandidate
type BecomeCandidatePayload struct {
	URL string
}

// FromBytes converts bytes to payload.
func (payload *BecomeCandidatePayload) FromBytes(b []byte) error {
	payloadPb := &dpospb.BecomeCandidatePayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.URL = payloadPb.Url
	return nil
}

// ToBytes returns marshaled BecomeCandidatePayload
func (payload *BecomeCandidatePayload) ToBytes() ([]byte, error) {
	payloadPb := &dpospb.BecomeCandidatePayload{
		Url: payload.URL,
	}
	return proto.Marshal(payloadPb)
}

// BecomeCandidateTx is a structure for quiting cadidate
type BecomeCandidateTx struct {
	*coreState.Transaction
	txHash        []byte
	candidateAddr common.Address
	collateral    *util.Uint128
	payload       *BecomeCandidatePayload
	size          int
}

var _ core.ExecutableTx = &BecomeCandidateTx{}

// NewBecomeCandidateTx returns BecomeCandidateTx
func NewBecomeCandidateTx(tx *coreState.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(BecomeCandidatePayload)
	if err := BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}

	return &BecomeCandidateTx{
		Transaction:   tx,
		txHash:        tx.Hash(),
		candidateAddr: tx.From(),
		collateral:    tx.Value(),
		payload:       payload,
		size:          size,
	}, nil
}

// Execute NewBecomeCandidateTx
func (tx *BecomeCandidateTx) Execute(b *core.Block) error {
	acc, err := b.State().GetAccount(tx.candidateAddr)
	if err != nil {
		return err
	}

	if acc.CandidateID != nil {
		return ErrAlreadyCandidate
	}
	_, err = acc.GetData("", []byte(coreState.AliasKey))
	if err == trie.ErrNotFound {
		return ErrAliasNotExist
	}

	minimumCollateral, err := util.NewUint128FromString(CandidateCollateralMinimum)
	if err != nil {
		return err
	}
	if tx.collateral.Cmp(minimumCollateral) < 0 {
		return ErrNotEnoughCandidateCollateral
	}

	// Subtract collateral from balance
	acc.Balance, err = acc.Balance.Sub(tx.collateral)
	if err == util.ErrUint128Underflow {
		return ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}

	acc.CandidateID = tx.txHash

	if err := b.State().PutAccount(acc); err != nil {
		return nil
	}

	// TODO: URL 유효성 확인? Regex? @shwankim

	candidate := &dState.Candidate{
		ID:         tx.txHash,
		Addr:       tx.candidateAddr,
		Collateral: tx.collateral,
		VotePower:  util.NewUint128(),
		URL:        tx.payload.URL,
		Timestamp:  b.Timestamp(),
	}

	// Add candidate to candidate state
	return b.State().DposState().PutCandidate(tx.txHash, candidate)
}

// Bandwidth returns bandwidth.
func (tx *BecomeCandidateTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1000, uint64(tx.size))
}

func (tx *BecomeCandidateTx) PointModifier(points *util.Uint128) (modifiedPoints *util.Uint128, err error) {
	return points, nil
}

func (tx *BecomeCandidateTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}

// QuitCandidateTx is a structure for quiting candidate
type QuitCandidateTx struct {
	*coreState.Transaction
	candidateAddr common.Address
	size          int
}

var _ core.ExecutableTx = &QuitCandidateTx{}

// NewQuitCandidateTx returns QuitCandidateTx
func NewQuitCandidateTx(tx *coreState.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, ErrInvalidAddress
	}

	return &QuitCandidateTx{
		Transaction:   tx,
		candidateAddr: tx.From(),
		size:          size,
	}, nil
}

// Execute QuitCandidateTx
func (tx *QuitCandidateTx) Execute(b *core.Block) error {
	acc, err := b.State().GetAccount(tx.candidateAddr)
	if err != nil {
		return err
	}
	if acc.CandidateID == nil {
		return ErrNotCandidate
	}

	candidate, err := b.State().DposState().GetCandidate(acc.CandidateID)
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
		return err
	}

	// change quit flag on candidate
	if err := b.State().DposState().DelCandidate(cID); err != nil {
		return err
	}

	return nil
}

// Bandwidth returns bandwidth.
func (tx *QuitCandidateTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1000, uint64(tx.size))
}

// PointModifier returns modifier
func (tx *QuitCandidateTx) PointModifier(points *util.Uint128) (modifiedPoints *util.Uint128, err error) {
	return points, nil
}

func (tx *QuitCandidateTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}
