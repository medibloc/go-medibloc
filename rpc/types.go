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
	ErrMsgAccountNotFound            = "account not found"
	ErrMsgBlockNotFound              = "block not found"
	ErrMsgBuildTransactionFail       = "cannot build transaction"
	ErrMsgConvertBlockFailed         = "cannot convert block"
	ErrMsgConvertBlockHeightFailed   = "cannot convert block height into integer"
	ErrMsgConvertBlockResponseFailed = "cannot convert block response"
	ErrMsgConvertTxResponseFailed    = "cannot convert transaction response"
	ErrMsgGetTransactionFailed       = "cannot get transaction from state"
	ErrMsgInvalidBlockHeight         = "invalid block height"
	ErrMsgInvalidDataType            = "invalid transaction data type"
	ErrMsgInvalidTransaction         = "invalid transaction"
	ErrMsgInvalidTxValue             = "invalid transaction value"
	ErrMsgInvalidTxDataPayload       = "invalid transaction data payload"
	ErrMsgTransactionNotFound        = "transaction not found"
	ErrMsgUnmarshalTransactionFailed = "cannot unmarshal transaction"
)
