// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

// Package storage implements Backend for BlockChain State
package storage

import "errors"

// ErrKeyNotFound entry not found error.
var ErrKeyNotFound = errors.New("not found")

// Storage interface of Storage.
type Storage interface {
	// Delete delete the key entry in Storage.
	Delete(key []byte) error

	// Get return the value to the key in Storage.
	Get(key []byte) ([]byte, error)

	// Put put the key-value entry to Storage.
	Put(key []byte, value []byte) error
}
