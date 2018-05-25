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
	"crypto/aes"
	"testing"

	"github.com/medibloc/go-medibloc/crypto/rand"
	"github.com/stretchr/testify/assert"
)

func TestAESDecryption(t *testing.T) {
	key := make([]byte, 32)
	in := make([]byte, 32)
	iv := rand.GetEntropyCSPRNG(aes.BlockSize)

	out, err := AESCTRXOR(key, in, iv)
	assert.Nil(t, err)
	assert.NotEqual(t, in, out, "Input and encrypted bytes should not equal")
	dec, err := AESCTRXOR(key, out, iv)
	assert.Nil(t, err)

	assert.Equal(t, in, dec, "Input and decrypted bytes should equal")
}
