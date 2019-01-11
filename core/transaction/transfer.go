package transaction

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
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

//TransferTx is a structure for sending MED
type TransferTx struct {
	from    common.Address
	to      common.Address
	value   *util.Uint128
	payload *DefaultPayload
	size    int
}

var _ core.ExecutableTx = &TransferTx{}

//NewTransferTx returns TransferTx
func NewTransferTx(tx *coreState.Transaction) (core.ExecutableTx, error) {
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
		from:    tx.From(),
		to:      tx.To(),
		value:   tx.Value(),
		payload: payload,
		size:    size,
	}, nil

}

//Execute TransferTx
func (tx *TransferTx) Execute(bs *core.BlockState) error {
	// subtract balance from sender's account
	sender, err := bs.GetAccount(tx.from)
	if err != nil {
		return err
	}
	sender.Balance, err = sender.Balance.Sub(tx.value)
	if err == util.ErrUint128Underflow {
		return ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}
	err = bs.PutAccount(sender)
	if err != nil {
		return err
	}

	// add balance to receiver's account
	receiver, err := bs.GetAccount(tx.to)
	if err != nil {
		return err
	}
	receiver.Balance, err = receiver.Balance.Add(tx.value)
	if err != nil {
		return err
	}
	err = bs.PutAccount(receiver)
	if err != nil {
		return err
	}
	return nil
}

//Bandwidth returns bandwidth.
func (tx *TransferTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1000, uint64(tx.size))
}

func (tx *TransferTx) PointModifier(points *util.Uint128) (modifiedPoints *util.Uint128, err error) {
	return points, nil
}
