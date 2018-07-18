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

package keystore_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/stretchr/testify/require"
)

func TestKeyStore(t *testing.T) {
	ks := keystore.NewKeyStore()

	key, err := crypto.GenerateKey(algorithm.SECP256K1)
	require.NoError(t, err)

	a, err := ks.SetKey(key)
	require.NoError(t, err)

	require.True(t, ks.HasAddress(a))

	err = ks.Delete(a)
	require.NoError(t, err)

	require.False(t, ks.HasAddress(a))
}
