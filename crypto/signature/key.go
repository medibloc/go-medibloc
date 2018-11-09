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

package signature

import (
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

const (
	// version compatible with ethereum, the version start with 3
	version = 3
)

// Key interface
type Key interface {

	// Algorithm returns the standard algorithm for this key. For
	// example, "ECDSA" would indicate that this key is a ECDSA key.
	Algorithm() algorithm.CryptoAlgorithm

	// Encoded returns the key in its primary encoding format, or null
	// if this key does not support encoding.
	Encoded() ([]byte, error)

	// Decode decode data to key
	Decode(data []byte) error

	// Clear clear key content
	Clear()
}

// PrivateKey privatekey interface
type PrivateKey interface {

	// Algorithm returns the standard algorithm for this key. For
	// example, "ECDSA" would indicate that this key is a ECDSA key.
	Algorithm() algorithm.CryptoAlgorithm

	// Encoded returns the key in its primary encoding format, or null
	// if this key does not support encoding.
	Encoded() ([]byte, error)

	// Decode decode data to key
	Decode(data []byte) error

	// Clear clear key content
	Clear()

	// PublicKey returns publickey
	PublicKey() PublicKey
}

// PublicKey publickey interface
type PublicKey interface {

	// Algorithm returns the standard algorithm for this key. For
	// example, "ECDSA" would indicate that this key is a ECDSA key.
	Algorithm() algorithm.CryptoAlgorithm

	// Encoded returns the key in its primary encoding format, or null
	// if this key does not support encoding.
	Encoded() ([]byte, error)

	// Decode decode data to key
	Decode(data []byte) error

	Compressed() ([]byte, error)

	Decompress(data []byte) error

	// Clear clear key content
	Clear()
}
