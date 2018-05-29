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
