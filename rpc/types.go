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

package rpc

// Block alias
const (
	// genesis block
	GENESIS = "genesis"
	// last irreversible block
	CONFIRMED = "confirmed"
	// tail block
	TAIL = "tail"
)

const (
	ErrMsgBlockNotFound           = "block not found"
	ErrMsgBuildTransactionFail    = "cannot build transaction"
	ErrMsgGetTransactionFailed    = "cannot get transaction from state"
	ErrMsgInternalError           = "unverified server error"
	ErrMsgInvalidBlockHeight      = "invalid block height"
	ErrMsgInvalidTransaction      = "invalid transaction"
	ErrMsgInvalidTxHash           = "invalid transaction hash"
	ErrMsgInvalidTxValue          = "invalid transaction value"
	ErrMsgTransactionNotFound     = "transaction not found"
	ErrMsgInvalidRequest          = "invalid request"
	ErrMsgFailedToUpdateBandwidth = "failed to update bandwidth"
	ErrMsgFailedToUpdateUnstaking = "failed to update Unstaking"
	ErrMsgTooManyBlocksRequest    = "exceed max blocks count"
	ErrMsgAliasNotFound           = "alias not found"
	ErrMsgInvalidBlockType        = "invalid block type"
	ErrMsgCandidateNotFound       = "candidate not found"
)

// Limit for get blocks request
const (
	MaxBlocksCount = 50
)
