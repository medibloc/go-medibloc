// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

package util

import (
	"errors"
	"math/big"
)

const (
	// Uint128Bytes defines the number of bytes for Uint128 type.
	Uint128Bytes = 16

	// Uint128Bits defines the number of bits for Uint128 type.
	Uint128Bits = 128
)

var (
	// ErrUint128Overflow indicates the value is greater than uint128 maximum value 2^128.
	ErrUint128Overflow = errors.New("uint128: overflow")

	// ErrUint128Underflow indicates the value is smaller then uint128 minimum value 0.
	ErrUint128Underflow = errors.New("uint128: underflow")

	// ErrUint128InvalidBytesSize indicates the bytes size is not equal to Uint128Bytes.
	ErrUint128InvalidBytesSize = errors.New("uint128: invalid bytes")
)

// Uint128 defines uint128 type, based on big.Int.
//
// For arithmetic operations, use uint128.Int.Add()/Sub()/Mul()/Div()/etc.
// For example, u1.Add(u1.Int, u2.Int) sets u1 to u1 + u2.
type Uint128 struct {
	value *big.Int
}

// Validate returns error if u is not a valid uint128, otherwise returns nil.
func (u *Uint128) Validate() error {
	if u.value.Sign() < 0 {
		return ErrUint128Underflow
	}
	if u.value.BitLen() > Uint128Bits {
		return ErrUint128Overflow
	}
	return nil
}

// NewUint128 returns a new Uint128 struct with default value.
func NewUint128() *Uint128 {
	return &Uint128{big.NewInt(0)}
}

// NewUint128FromInt returns a new Uint128 struct with given value and have a check.
func NewUint128FromInt(i int64) (*Uint128, error) {
	obj := &Uint128{big.NewInt(i)}
	if err := obj.Validate(); nil != err {
		return nil, err
	}
	return obj, nil
}

// ToFixedSizeBytes converts Uint128 to Big-Endian fixed size bytes.
func (u *Uint128) ToFixedSizeBytes() ([16]byte, error) {
	var res [16]byte
	if err := u.Validate(); err != nil {
		return res, err
	}
	bs := u.value.Bytes()
	l := len(bs)
	if l == 0 {
		return res, nil
	}
	idx := Uint128Bytes - len(bs)
	if idx < Uint128Bytes {
		copy(res[idx:], bs)
	}
	return res, nil
}

// ToFixedSizeByteSlice converts Uint128 to Big-Endian fixed size byte slice.
func (u *Uint128) ToFixedSizeByteSlice() ([]byte, error) {
	bytes, err := u.ToFixedSizeBytes()
	return bytes[:], err
}

// String returns the string representation of x.
func (u *Uint128) String() string {
	return u.value.Text(10)
}

// FromFixedSizeBytes converts Big-Endian fixed size bytes to Uint128.
func (u *Uint128) FromFixedSizeBytes(bytes [16]byte) *Uint128 {
	u.FromFixedSizeByteSlice(bytes[:])
	return u
}

// FromFixedSizeByteSlice converts Big-Endian fixed size bytes to Uint128.
func (u *Uint128) FromFixedSizeByteSlice(bytes []byte) (*Uint128, error) {
	if len(bytes) != Uint128Bytes {
		return nil, ErrUint128InvalidBytesSize
	}
	i := 0
	for ; i < Uint128Bytes; i++ {
		if bytes[i] != 0 {
			break
		}
	}
	if i < Uint128Bytes {
		u.value.SetBytes(bytes[i:])
	} else {
		u.value.SetUint64(0)
	}
	return u, nil
}

//Add returns u + x
func (u *Uint128) Add(x *Uint128) (*Uint128, error) {
	obj := &Uint128{NewUint128().value.Add(u.value, x.value)}
	if err := obj.Validate(); nil != err {
		return u, err
	}
	return obj, nil
}

//Sub returns u - x
func (u *Uint128) Sub(x *Uint128) (*Uint128, error) {
	obj := &Uint128{NewUint128().value.Sub(u.value, x.value)}
	if err := obj.Validate(); nil != err {
		return u, err
	}
	return obj, nil
}

//Mul returns u * x
func (u *Uint128) Mul(x *Uint128) (*Uint128, error) {
	obj := &Uint128{NewUint128().value.Mul(u.value, x.value)}
	if err := obj.Validate(); nil != err {
		return u, err
	}
	return obj, nil
}

//Div returns u / x
func (u *Uint128) Div(x *Uint128) (*Uint128, error) {
	obj := &Uint128{NewUint128().value.Div(u.value, x.value)}
	if err := obj.Validate(); nil != err {
		return u, err
	}
	return obj, nil
}

//Exp returns u^x
func (u *Uint128) Exp(x *Uint128) (*Uint128, error) {
	obj := &Uint128{NewUint128().value.Exp(u.value, x.value, nil)}
	if err := obj.Validate(); nil != err {
		return u, err
	}
	return obj, nil
}

//DeepCopy returns a deep copy of u
func (u *Uint128) DeepCopy() *Uint128 {
	z := new(big.Int)
	z.Set(u.value)
	return &Uint128{z}
}

// Cmp compares u and x and returns:
//
//   -1 if u <  x
//    0 if u == x
//   +1 if u >  x
func (u *Uint128) Cmp(x *Uint128) int {
	return u.value.Cmp(x.value)
}

//Bytes absolute value of u as a big-endian byte slice.
func (u *Uint128) Bytes() []byte {
	return u.value.Bytes()
}
