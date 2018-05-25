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
