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

package core_test

import (
	"encoding/hex"
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/stretchr/testify/assert"
)

func TestTrieBatch(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	trieBatch, err := core.NewTrieBatch(nil, s)
	assert.Nil(t, err)

	err = trieBatch.BeginBatch()
	assert.Nil(t, err)

	err = trieBatch.BeginBatch()
	assert.Equal(t, core.ErrBeginAgainInBatch, err)

	key1, _ := hex.DecodeString("2ed1b10c")
	val1 := []byte("leafmedi1")
	val2 := []byte("leafmedi2")

	testGetValue := func (key []byte, val []byte) {
		getVal, err := trieBatch.Get(key1)
		if val == nil {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, val, getVal)
		}
	}

	err = trieBatch.Put(key1, val1)
	assert.Nil(t, err)
	testGetValue(key1, val1)

	err = trieBatch.Commit()
	assert.Nil(t, err)

	err = trieBatch.BeginBatch()
	assert.Nil(t, err)

	err = trieBatch.Put(key1, val2)
	assert.Nil(t, err)
	trieBatch.RollBack()
	testGetValue(key1, val1)

	err = trieBatch.BeginBatch()
	assert.Nil(t, err)

	err = trieBatch.Delete(key1)
	assert.Nil(t, err)
	testGetValue(key1, nil)
	trieBatch.RollBack()
	testGetValue(key1, val1)
}
