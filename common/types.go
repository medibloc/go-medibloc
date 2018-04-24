package common

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// Types' length.
const (
	HashLength    = 32
	AddressLength = 33
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// BytesToHash converts bytes to Hash.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// BigToHash converts big Int to Hash.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash converts hex string to Hash.
func HexToHash(s string) Hash { return BytesToHash(byteutils.FromHex(s)) }

// Str returns Hash in string format.
func (h Hash) Str() string { return string(h[:]) }

// Bytes returns Hash in bytes format.
func (h Hash) Bytes() []byte { return h[:] }

// Hex returns Hash in hex string
func (h Hash) Hex() string { return byteutils.Bytes2Hex(h[:]) }

// SetBytes set bytes to Hash.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Equals compare Hash.
func (h Hash) Equals(b Hash) bool {
	return bytes.Compare(h[:], b[:]) == 0
}

// IsZeroHash checks if hash h is zero hash (0x00000...)
func IsZeroHash(h Hash) bool {
	return h == Hash{}
}

// ZeroHash returns hash with zero value
func ZeroHash() Hash {
	return Hash{}
}

// Address represents Address.
type Address [AddressLength]byte

// BytesToAddress gets Address from bytes.
func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// BigToAddress gets Address from big Int.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress gets Address from hex string.
func HexToAddress(s string) Address { return BytesToAddress(byteutils.FromHex(s)) }

// IsHexAddress checks hex address.
func IsHexAddress(s string) bool {
	if byteutils.HasHexPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && byteutils.IsHex(s)
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
		return Address{}, errors.New("Invalid public key algorithm")
	}
}

// AddressToPublicKey gets PublicKey from Address.
func AddressToPublicKey(addr Address, alg algorithm.Algorithm) (signature.PublicKey, error) {
	switch alg {
	case algorithm.SECP256K1:
		var pubKey *secp256k1.PublicKey
		if err := pubKey.Decompress(addr.Bytes()); err != nil {
			return nil, err
		}
		return pubKey, nil
	default:
		return nil, errors.New("Invalid public key algorithm")
	}
}

// Str returns Address in string.
func (a Address) Str() string { return string(a[:]) }

// Bytes returns Address in bytes.
func (a Address) Bytes() []byte { return a[:] }

// Big returns Address in big Int.
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }

// Hash returns Address in Hash.
func (a Address) Hash() Hash { return BytesToHash(a[:]) }

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
