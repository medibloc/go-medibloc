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

func testBatch(t *testing.T, batch core.Batch) {
	err := batch.BeginBatch()
	assert.Nil(t, err)

	err = batch.BeginBatch()
	assert.Equal(t, core.ErrBeginAgainInBatch, err)

	err = batch.RollBack()
	assert.Nil(t, err)

	err = batch.RollBack()
	assert.Equal(t, core.ErrNotBatching, err)
}

func TestTrieBatch(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	trieBatch, err := core.NewTrieBatch(nil, s)
	assert.Nil(t, err)

	testBatch(t, trieBatch)

	err = trieBatch.BeginBatch()
	assert.Nil(t, err)

	err = trieBatch.BeginBatch()
	assert.Equal(t, core.ErrBeginAgainInBatch, err)

	key1, _ := hex.DecodeString("2ed1b10c")
	val1 := []byte("leafmedi1")
	val2 := []byte("leafmedi2")

	testGetValue := func(key []byte, val []byte) {
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

func TestAccountState(t *testing.T) {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)

	asBatch, err := core.NewAccountStateBatch(s)
	assert.Nil(t, err)
	assert.NotNil(t, asBatch)

	testBatch(t, asBatch)

	err = asBatch.BeginBatch()
	assert.Nil(t, err)

	acc1Address, _ := hex.DecodeString("account1")

	checkBalance := func (address []byte, expected uint64) {
		as := asBatch.AccountState()
		acc, err := as.GetAccount(address)
		assert.Nil(t, err)
		assert.Equal(t, expected, acc.Balance())
	}

	amount := uint64(1)

	err = asBatch.SubBalance(acc1Address, amount)
	assert.Equal(t, core.ErrBalanceNotEnough, err)
	err = asBatch.AddBalance(acc1Address, amount)
	assert.Nil(t, err)

	acc, err := asBatch.GetAccount(acc1Address)
	assert.Nil(t, err)
	assert.Equal(t, acc1Address, acc.Address())

	err = asBatch.RollBack()
	assert.Nil(t, err)

	err = asBatch.BeginBatch()
	assert.Nil(t, err)

	_, err = asBatch.GetAccount(acc1Address)
	assert.NotNil(t, err)

	err = asBatch.AddBalance(acc1Address, amount)
	assert.Nil(t, err)

	err = asBatch.Commit()
	assert.Nil(t, err)

	err = asBatch.BeginBatch()
	assert.Nil(t, err)

	err = asBatch.AddBalance(acc1Address, amount)
	assert.Nil(t, err)

	acc, err = asBatch.GetAccount(acc1Address)
	assert.Equal(t, 2 * amount, acc.Balance())
	checkBalance(acc1Address, amount)

	err = asBatch.Commit()
	assert.Nil(t, err)

	checkBalance(acc1Address, 2 * amount)
}
