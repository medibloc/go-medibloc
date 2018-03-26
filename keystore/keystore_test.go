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

package keystore_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/keystore"
)

var testSigData = make([]byte, 32)

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

func TestKeyStore(t *testing.T) {
	ks := keystore.NewKeyStore()

	key, err := crypto.GenerateKey(keystore.SECP256K1)
	if err != nil {
		t.Fatal(err)
	}

	a, err := ks.SetKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if !ks.HasAddress(a) {
		t.Errorf("HasAccount(%x) should've returned true", a)
	}
	if err := ks.Delete(a); err != nil {
		t.Errorf("Delete error: %v", err)
	}
	if ks.HasAddress(a) {
		t.Errorf("HasAccount(%x) should've returned true after Delete", a)
	}
}
