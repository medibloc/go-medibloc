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

package secp256k1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/medibloc/go-medibloc/util/math"
)

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1Halfn = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

// NewECDSAPrivateKey generate a ecdsa private key
func NewECDSAPrivateKey() *ecdsa.PrivateKey {
	var priv *ecdsa.PrivateKey
	for {
		priv, _ = ecdsa.GenerateKey(S256(), rand.Reader)
		if SeckeyVerify(FromECDSAPrivateKey(priv)) {
			break
		}
	}
	return priv
}

// FromECDSAPrivateKey exports a private key into a binary dump.
func FromECDSAPrivateKey(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

// HexToECDSA gets a private key from hex string.
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return ToECDSAPrivateKey(b)
}

// ToECDSAPrivateKey gets a private key from bytes.
func ToECDSAPrivateKey(d []byte) (*ecdsa.PrivateKey, error) {
	return toECDSAPrivateKey(d, true)
}

// ToECDSAPrivateKeyUnsafe is ToECDSAPrivateKey's unsafe function.
func ToECDSAPrivateKeyUnsafe(d []byte) *ecdsa.PrivateKey {
	priv, _ := toECDSAPrivateKey(d, false)
	return priv
}

func toECDSAPrivateKey(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

// FromECDSAPublicKey exports a public key into a binary dump.
func FromECDSAPublicKey(pub *ecdsa.PublicKey) ([]byte, error) {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil, errors.New("invalid public key input")
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y), nil
}

// ToECDSAPublicKey creates a public key with the given data value.
func ToECDSAPublicKey(pub []byte) (*ecdsa.PublicKey, error) {
	if len(pub) == 0 {
		return nil, errors.New("invalid public key")
	}
	x, y := elliptic.Unmarshal(S256(), pub)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
}

// zeroKey zeroes the private key
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}
