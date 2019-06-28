package transaction

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/util"
)

// DefaultPayload is payload type for any type of message
type DefaultPayload struct {
	Message string
}

// FromBytes converts bytes to payload.
func (payload *DefaultPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.DefaultPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.Message = payloadPb.Message
	return nil
}

// ToBytes returns marshaled DefaultPayload
func (payload *DefaultPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.DefaultPayload{
		Message: payload.Message,
	}
	return proto.Marshal(payloadPb)
}

// TransferTx is a structure for sending MED
type TransferTx struct {
	*core.Transaction
	from    common.Address
	to      common.Address
	value   *util.Uint128
	payload *DefaultPayload
	size    int
}

var _ core.ExecutableTx = &TransferTx{}

// NewTransferTx returns TransferTx
func NewTransferTx(tx *core.Transaction) (core.ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	if tx.Value().Cmp(util.Uint128Zero()) == 0 {
		return nil, ErrVoidTransaction
	}
	payload := new(DefaultPayload)
	if err := BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) || !common.IsHexAddress(tx.To().Hex()) {
		return nil, ErrInvalidAddress
	}

	return &TransferTx{
		Transaction: tx,
		from:        tx.From(),
		to:          tx.To(),
		value:       tx.Value(),
		payload:     payload, // TODO not used
		size:        size,
	}, nil

}

// Execute TransferTx
func (tx *TransferTx) Execute(b *core.Block) error {
	// subtract balance from sender's account
	sender, err := b.State().GetAccount(tx.from)
	if err != nil {
		return err
	}
	sender.Balance, err = sender.Balance.Sub(tx.value)
	if err == util.ErrUint128Underflow {
		return core.ErrNotEnoughBalance
	}
	if err != nil {
		return err
	}
	err = b.State().PutAccount(sender)
	if err != nil {
		return err
	}

	// add balance to receiver's account
	receiver, err := b.State().GetAccount(tx.to)
	if err != nil {
		return err
	}
	receiver.Balance, err = receiver.Balance.Add(tx.value)
	if err != nil {
		return err
	}
	err = b.State().PutAccount(receiver)
	if err != nil {
		return err
	}
	return nil
}

// Bandwidth returns bandwidth.
func (tx *TransferTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1000, uint64(tx.size))
}

// PointChange returns account's point change when applying this transaction.
func (tx *TransferTx) PointChange() (neg bool, abs *util.Uint128) {
	return false, util.Uint128Zero()
}

// RecoverFrom returns from account's address.
func (tx *TransferTx) RecoverFrom() (common.Address, error) {
	return recoverSigner(tx.Transaction)
}
