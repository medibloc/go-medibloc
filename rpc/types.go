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

// Error response strings of APIService
const (
	ErrMsgBlockNotFound              = "block not found"
	ErrMsgBuildTransactionFail       = "cannot build transaction"
	ErrMsgConvertAccountFailed       = "cannot convert accout"
	ErrMsgConvertBlockFailed         = "cannot convert block"
	ErrMsgConvertBlockHeightFailed   = "cannot convert block height into integer"
	ErrMsgConvertBlockResponseFailed = "cannot convert block response"
	ErrMsgConvertTxResponseFailed    = "cannot convert transaction response"
	ErrMsgGetTransactionFailed       = "cannot get transaction from state"
	ErrMsgInternalError              = "unverified server error"
	ErrMsgInvalidBlockHeight         = "invalid block height"
	ErrMsgInvalidDataType            = "invalid transaction data type"
	ErrMsgInvalidTransaction         = "invalid transaction"
	ErrMsgInvalidTxHash              = "invalid transaction hash"
	ErrMsgInvalidTxValue             = "invalid transaction value"
	ErrMsgInvalidTxDataPayload       = "invalid transaction data payload"
	ErrMsgTransactionNotFound        = "transaction not found"
	ErrMsgUnmarshalTransactionFailed = "cannot unmarshal transaction"
	ErrMsgInvalidRequest             = "invalid request"
	ErrMsgFailedToUpdateBandwidth    = "failed to update bandwidth"
	ErrMsgFailedToUpdateUnstaking    = "failed to update Unstaking"
	ErrMsgTooManyBlocksRequest       = "exceed max blocks count"
)

// Limit for get blocks request
const (
	MaxBlocksCount = 50
)
