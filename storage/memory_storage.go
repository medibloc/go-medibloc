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

package storage

import (
	"encoding/hex"
	"sync"
)

// MemoryStorage memory storage
type MemoryStorage struct {
	data *sync.Map
}

// NewMemoryStorage init a storage
func NewMemoryStorage() (*MemoryStorage, error) {
	return &MemoryStorage{
		data: new(sync.Map),
	}, nil
}

// Delete delete the key entry in Storage.
func (s *MemoryStorage) Delete(key []byte) error {
	s.data.Delete(hex.EncodeToString(key))
	return nil
}

// Get return the value to the key in Storage.
func (s *MemoryStorage) Get(key []byte) ([]byte, error) {
	if entry, ok := s.data.Load(hex.EncodeToString(key)); ok {
		return entry.([]byte), nil
	}
	return nil, ErrKeyNotFound
}

// Put put the key-value entry to Storage.
func (s *MemoryStorage) Put(key []byte, value []byte) error {
	s.data.Store(hex.EncodeToString(key), value)
	return nil
}
