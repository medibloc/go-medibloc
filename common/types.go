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

package common

import (
	"bytes"
	"errors"
	"math/big"
	"unicode"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// Types' length.
const (
	AddressLength = 33
	HashLength    = 32
)

// Address represents Address.
type Address [AddressLength]byte

// BytesToAddress gets Address from bytes.
func BytesToAddress(b []byte) Address {
	var a Address
	a.FromBytes(b)
	return a
}

// HexToAddress gets Address from hex string.
func HexToAddress(s string) (Address, error) {
	addr, err := byteutils.FromHex(s)
	return BytesToAddress(addr), err
}

// IsHexAddress checks hex address.
func IsHexAddress(s string) bool {
	if byteutils.HasHexPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && byteutils.IsHex(s)
}

// IsHash checks hash string.
func IsHash(s string) bool {
	return len(s) == 2*HashLength && byteutils.IsHex(s)
}

// PublicKeyToAddress gets Address from PublicKey.
func PublicKeyToAddress(p signature.PublicKey) (Address, error) {
	switch p.Algorithm() {
	case algorithm.SECP256K1:
		buf, err := p.Compressed()
		if err != nil {
			return Address{}, err
		}
		return BytesToAddress(buf), nil
	default:
		return Address{}, algorithm.ErrInvalidCryptoAlgorithm
	}
}

// Str returns Address in string.
func (a Address) Str() string { return string(a[:]) }

// Bytes returns Address in bytes.
func (a Address) Bytes() []byte { return a[:] }

// Big returns Address in big Int.
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hex returns Address in hex string
func (a Address) Hex() string { return byteutils.Bytes2Hex(a[:]) }

// SetBytes the address to the value of b. If b is larger than len(a) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// Equals compare Address.
func (a Address) Equals(b Address) bool {
	return bytes.Compare(a[:], b[:]) == 0
}

// String is a stringer interface of Address.
func (a Address) String() string {
	return byteutils.Bytes2Hex(a.Bytes())
}

//ToBytes convert address to byte slice (for trie.Serializable)
func (a Address) ToBytes() ([]byte, error) {
	return a.Bytes(), nil
}

//FromBytes convert byte slice to slice (for trie.Serializable)
func (a *Address) FromBytes(b []byte) error {
	a.SetBytes(b)
	return nil
}

const (
	//AliasKey key for find aliasname
	AliasKey = "alias"
	//AliasMaxLength is the max length of alias
	AliasMaxLength = 12
)

// Error types for check alias
var (
	ErrAliasEmptyString = errors.New("aliasname should not be empty string")
	ErrAliasLengthLimit = errors.New("aliasname should not be longer than 12 letters")
	ErrAliasInvalidChar = errors.New("aliasname should contain only lowercase letters and numbers")
	ErrAliasFirstLetter = errors.New("first letter of alias name should not be a number")
)

// ValidateAlias checks alias
func ValidateAlias(alias string) error {
	if alias == "" {
		return ErrAliasEmptyString
	}
	if len(alias) > AliasMaxLength {
		return ErrAliasLengthLimit
	}
	for i := 0; i < len(alias); i++ {
		ch := rune(alias[i])

		if !(unicode.IsNumber(ch) || unicode.IsLower(ch)) {
			return ErrAliasInvalidChar
		}
		if i == 0 && unicode.IsNumber(ch) {
			return ErrAliasFirstLetter
		}
	}
	return nil
}
