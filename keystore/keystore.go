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

package keystore

import (
	"errors"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/crypto/signature"
)

var (
	// ErrNoMatch address key doesn't match error.
	ErrNoMatch = errors.New("no key for given address")
)

// KeyStore manages private keys.
type KeyStore struct {
	keys map[common.Address]signature.PrivateKey
}

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore() *KeyStore {
	ks := &KeyStore{}
	ks.keys = make(map[common.Address]signature.PrivateKey)
	return ks
}

// SetKey set key.
func (ks *KeyStore) SetKey(key signature.PrivateKey) (common.Address, error) {
	addr, err := common.PublicKeyToAddress(key.PublicKey())
	if err != nil {
		return common.Address{}, err
	}
	ks.keys[addr] = key
	return addr, nil
}

// Delete deletes key.
func (ks *KeyStore) Delete(a common.Address) error {
	if !ks.HasAddress(a) {
		return ErrNoMatch
	}
	delete(ks.keys, a)
	return nil
}

// HasAddress reports whether a key with the given address is present.
func (ks *KeyStore) HasAddress(addr common.Address) bool {
	return ks.keys[addr] != nil
}

// Accounts returns all key files present in the directory.
func (ks *KeyStore) Accounts() []common.Address {
	addresses := []common.Address{}
	for addr := range ks.keys {
		addresses = append(addresses, addr)
	}
	return addresses
}

// GetKey gets key.
func (ks *KeyStore) GetKey(a common.Address) (signature.PrivateKey, error) {
	if !ks.HasAddress(a) {
		return nil, ErrNoMatch
	}
	return ks.keys[a], nil
}
