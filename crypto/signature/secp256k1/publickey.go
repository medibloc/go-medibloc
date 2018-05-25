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

	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

// PublicKey ecdsa publickey.
type PublicKey struct {
	publicKey ecdsa.PublicKey
}

// NewPublicKey generate PublicKey.
func NewPublicKey(pub ecdsa.PublicKey) *PublicKey {
	ecdsaPub := new(PublicKey)
	ecdsaPub.publicKey = pub
	return ecdsaPub
}

// Algorithm algorithm name.
func (k *PublicKey) Algorithm() algorithm.Algorithm {
	return algorithm.SECP256K1
}

// Encoded encoded to byte.
func (k *PublicKey) Encoded() ([]byte, error) {
	return FromECDSAPublicKey(&k.publicKey)
}

// Decode decode data to key.
func (k *PublicKey) Decode(b []byte) error {
	pub, err := ToECDSAPublicKey(b)
	if err != nil {
		return err
	}
	k.publicKey = *pub
	return nil
}

// Clear clear key content.
func (k *PublicKey) Clear() {
	k.publicKey = ecdsa.PublicKey{}
}

// Verify verify ecdsa publickey.
func (k *PublicKey) Verify(msg []byte, sig []byte) (bool, error) {
	pub, err := k.Encoded()
	if err != nil {
		return false, err
	}
	return VerifySignature(pub, msg, sig), nil
}

// Compressed encodes a public key to 33-byte compressed format.
func (k *PublicKey) Compressed() ([]byte, error) {
	return CompressPubkey(k.publicKey.X, k.publicKey.Y)
}

// Decompress parses a public key in the 33-byte compressed format.
func (k *PublicKey) Decompress(data []byte) error {
	x, y, err := DecompressPubkey(data)
	if err != nil {
		return err
	}

	k.publicKey = ecdsa.PublicKey{
		Curve: S256(), X: x, Y: y,
	}

	return nil
}
