package core

import (
	"github.com/medibloc/go-medibloc/util"
)

// RtWithdraw represents payload of for withdrawing vestings
type RtWithdraw struct {
	Amount *util.Uint128
}

// NewRtWithdraw generates a RtWithdraw
func NewRtWithdraw(amount *util.Uint128) (*RtWithdraw, error) {
	return &RtWithdraw{Amount: amount}, nil
}

// Serialize a RtWithdraw to a byte array
func (w *RtWithdraw) Serialize() ([]byte, error) {
	return w.Amount.ToFixedSizeByteSlice()
}

// Deserialize a byte array and get a RtWithdraw
func (w *RtWithdraw) Deserialize(b []byte) error {
	return w.Amount.FromFixedSizeByteSlice(b)
}
