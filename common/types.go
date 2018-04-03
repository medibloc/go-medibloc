package common

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"golang.org/x/crypto/sha3"
)

const (
	HashLength    = 32
	AddressLength = 20
)

type Hash [HashLength]byte

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }
func HexToHash(s string) Hash   { return BytesToHash(FromHex(s)) }

func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }

func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

func (h Hash) Equals(b Hash) bool {
	return bytes.Compare(h[:], b[:]) == 0
}

type Address [AddressLength]byte

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }
func HexToAddress(s string) Address   { return BytesToAddress(FromHex(s)) }

func IsHexAddress(s string) bool {
	if hasHexPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}

func PublicKeyToAddress(p signature.PublicKey) (Address, error) {
	switch p.Algorithm() {
	case algorithm.SECP256K1:
		buf, err := p.Encoded()
		if err != nil {
			return Address{}, err
		}
		hash := sha3.Sum256(buf[1:])
		return BytesToAddress(hash[12:]), nil
	default:
		return Address{}, errors.New("Invalid public key algorithm")
	}
}

func (a Address) Str() string   { return string(a[:]) }
func (a Address) Bytes() []byte { return a[:] }
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }
func (a Address) Hash() Hash    { return BytesToHash(a[:]) }

// Sets the address to the value of b. If b is larger than len(a) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

func (a Address) Equals(b Address) bool {
	return bytes.Compare(a[:], b[:]) == 0
}
