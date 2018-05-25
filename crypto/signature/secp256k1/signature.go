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
	"errors"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

// Signature signature ecdsa
type Signature struct {
	privateKey *PrivateKey
	publicKey  *PublicKey
}

// Algorithm secp256k1 algorithm
func (s *Signature) Algorithm() algorithm.Algorithm {
	return algorithm.SECP256K1
}

// InitSign ecdsa init sign
func (s *Signature) InitSign(priv signature.PrivateKey) {
	s.privateKey = priv.(*PrivateKey)
}

// Sign ecdsa sign
func (s *Signature) Sign(data []byte) ([]byte, error) {
	if s.privateKey == nil {
		return nil, errors.New("signature private key is nil")
	}

	sig, err := s.privateKey.Sign(data)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// RecoverPublic returns a public key
func (s *Signature) RecoverPublic(data []byte, sig []byte) (signature.PublicKey, error) {
	pub, err := RecoverPubkey(data, sig)
	if err != nil {
		return nil, err
	}
	pubKey, err := ToECDSAPublicKey(pub)
	if err != nil {
		return nil, err
	}

	return NewPublicKey(*pubKey), nil
}

// InitVerify ecdsa verify init
func (s *Signature) InitVerify(pub signature.PublicKey) {
	s.publicKey = pub.(*PublicKey)
}

// Verify ecdsa verify
func (s *Signature) Verify(data []byte, sig []byte) (bool, error) {
	if s.publicKey == nil {
		return false, errors.New("signature public key is nil")
	}

	return s.publicKey.Verify(data, sig)
}
