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

package crypto

import (
	"errors"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
)

var (
	// ErrAlgorithmInvalid invalid Algorithm for sign.
	ErrAlgorithmInvalid = errors.New("invalid Algorithm")
)

// GenerateKey generates PrivateKey.
func GenerateKey(alg algorithm.Algorithm) (signature.PrivateKey, error) {
	switch alg {
	case algorithm.SECP256K1:
		return secp256k1.GeneratePrivateKey(), nil
	default:
		return nil, ErrAlgorithmInvalid
	}
}

// NewSignature returns signature from algorithm.
func NewSignature(alg algorithm.Algorithm) (signature.Signature, error) {
	switch alg {
	case algorithm.SECP256K1:
		return new(secp256k1.Signature), nil
	default:
		return nil, ErrAlgorithmInvalid
	}
}

// CheckAlgorithm checks algorithm.
func CheckAlgorithm(alg algorithm.Algorithm) error {
	switch alg {
	case algorithm.SECP256K1:
		return nil
	default:
		return ErrAlgorithmInvalid
	}
}
