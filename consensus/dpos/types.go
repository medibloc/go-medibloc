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
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package dpos

import (
	"errors"
	"time"
)

// Transaction's related to dpos
const (
	TxOpBecomeCandidate = "become_candidate"
	TxOpQuitCandidacy   = "quit_candidacy"
	TxOpVote            = "vote"
)

// Consensus properties.
const (
	BlockInterval  = 3 * time.Second
	NumberOfRounds = 1

	miningTickInterval = 100 * time.Millisecond
	minMintDuration    = 500 * time.Millisecond
	maxMintDuration    = 1100 * time.Millisecond
)

// Error types of dpos package.
var (
	ErrAlreadyCandidate       = errors.New("account is already a candidate")
	ErrBlockMintedInNextSlot  = errors.New("cannot mint block now, there is a block minted in current slot")
	ErrInvalidBlockForgeTime  = errors.New("invalid time to forge block")
	ErrInvalidBlockInterval   = errors.New("invalid block interval")
	ErrInvalidBlockProposer   = errors.New("invalid block proposer")
	ErrInvalidBlockReward     = errors.New("invalid block reward")
	ErrInvalidDynastySize     = errors.New("invalid dynasty size")
	ErrNotCandidate           = errors.New("account is not a candidate")
	ErrNoVote                 = errors.New("vote to no one")
	ErrOverMaxVote            = errors.New("too many vote")
	ErrDuplicateVote          = errors.New("cannot vote multiple vote for same account")
	ErrWaitingBlockInLastSlot = errors.New("cannot mint block now, waiting for last block")
	ErrProposerConfigNotFound = errors.New("proposer config not found")
)
