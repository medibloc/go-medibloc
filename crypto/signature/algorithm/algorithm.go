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

package algorithm

import "errors"

// CryptoAlgorithm type alias
type CryptoAlgorithm uint8

// HashAlgorithm type alias
type HashAlgorithm uint8

const (
	// SECP256K1 a type of signer
	SECP256K1 CryptoAlgorithm = 1
	// SHA3256 a type of hash
	SHA3256 HashAlgorithm = 2
)

// Error types of algorithm package.
var (
	ErrInvalidHashAlgorithm   = errors.New("Invalid hash algorithm")
	ErrInvalidCryptoAlgorithm = errors.New("Invalid crypto algorithm")
)
