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

package byteutils

import (
	"encoding/binary"
	"encoding/hex"
)

// Encode encodes object to Encoder.
func Encode(s interface{}, enc Encoder) ([]byte, error) {
	return enc.EncodeToBytes(s)
}

// Decode decodes []byte from Decoder.
func Decode(data []byte, dec Decoder) (interface{}, error) {
	return dec.DecodeFromBytes(data)
}

// Uint64 encodes []byte.
func Uint64(data []byte) uint64 {
	return binary.BigEndian.Uint64(data)
}

// FromUint64 decodes unit64 value.
func FromUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// Uint32 encodes []byte.
func Uint32(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

// FromUint32 decodes uint32.
func FromUint32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// Uint16 encodes []byte.
func Uint16(data []byte) uint16 {
	return binary.BigEndian.Uint16(data)
}

// FromUint16 decodes uint16.
func FromUint16(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

// Int64 encodes []byte.
func Int64(data []byte) int64 {
	return int64(binary.BigEndian.Uint64(data))
}

// FromInt64 decodes int64 v.
func FromInt64(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// Int32 encodes []byte.
func Int32(data []byte) int32 {
	return int32(binary.BigEndian.Uint32(data))
}

// FromInt32 decodes int32 v.
func FromInt32(v int32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

// Int16 encode []byte.
func Int16(data []byte) int16 {
	return int16(binary.BigEndian.Uint16(data))
}

// FromInt16 decodes int16 v.
func FromInt16(v int16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

// Equal checks whether byte slice a and b are equal.
func Equal(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CopyBytes returns a copied clone of the bytes
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

// ToHex returns hex string of bytes
func ToHex(b []byte) string {
	hex := Bytes2Hex(b)
	if len(hex) == 0 {
		hex = "0"
	}

	return "0x" + hex
}

// FromHex returns decoded bytes array from hex string
func FromHex(s string) ([]byte, error) {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

// Bytes2Hex encodes bytes and returns hex string without '0x' prefix
func Bytes2Hex(d []byte) string {
	return hex.EncodeToString(d)
}

// Hex2Bytes decodes hex string without '0x' prefix and returns bytes
func Hex2Bytes(str string) ([]byte, error) {
	h, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// BytesSlice2HexSlice encodes bytes slice and returns hex string slice without '0x' prefix
func BytesSlice2HexSlice(d [][]byte) []string {
	var hs = make([]string, len(d))
	for i, b := range d {
		hs[i] = Bytes2Hex(b)
	}
	return hs
}

// HexSlice2BytesSlice decodes hex string slice without '0x' prefix and returns bytes slice
func HexSlice2BytesSlice(ss []string) ([][]byte, error) {
	var d = make([][]byte, len(ss))
	var err error
	for i, h := range ss {
		d[i], err = Hex2Bytes(h)
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

// RightPadBytes adds padding in right side
func RightPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded, slice)

	return padded
}

// LeftPadBytes adds padding in left side
func LeftPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}

// HasHexPrefix checks if hex string has '0x' or '0X' prefix
func HasHexPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// IsHex checks if a string is hex representation
func IsHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}
