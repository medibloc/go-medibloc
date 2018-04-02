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

func getStorage(t *testing.T) storage.Storage {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	return s
}

func TestTrieBatch(t *testing.T) {
	s := getStorage(t)

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

func TestTrieBatch_Clone(t *testing.T) {
	s := getStorage(t)

	trBatch1, err := core.NewTrieBatch(nil, s)
	assert.Nil(t, err)

	trBatch2, err := trBatch1.Clone()

	err = trBatch1.BeginBatch()
	assert.Nil(t, err)

	_, err = trBatch1.Clone()
	assert.Equal(t, core.ErrCannotCloneOnBatching, err)

	key1, _ := hex.DecodeString("2ed1b10c")
	val1 := []byte("leafmedi1")

	err = trBatch1.Put(key1, val1)
	assert.Nil(t, err)

	err = trBatch1.Commit()
	assert.Nil(t, err)

	_, err = trBatch1.Get(key1)
	assert.Nil(t, err)
	_, err = trBatch2.Get(key1)
	assert.Equal(t, core.ErrNotFound, err)

	err = trBatch2.BeginBatch()
	assert.Nil(t, err)

	err = trBatch2.Put(key1, val1)
	assert.Nil(t, err)

	err = trBatch2.Commit()
	assert.Nil(t, err)
	getVal1, err := trBatch1.Get(key1)
	assert.Nil(t, err)
	getVal2, err := trBatch2.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, getVal1, getVal2)
}

func getAddress() []byte {
	acc1Address, _ := hex.DecodeString("account1")
	return acc1Address
}

func TestAccountState(t *testing.T) {
	s := getStorage(t)

	asBatch, err := core.NewAccountStateBatch(nil, s)
	assert.Nil(t, err)
	assert.NotNil(t, asBatch)

	testBatch(t, asBatch)

	err = asBatch.BeginBatch()
	assert.Nil(t, err)

	acc1Address := getAddress()
	amount := uint64(1)

	checkBalance := func(address []byte, expected uint64) {
		as := asBatch.AccountState()
		acc, err := as.GetAccount(address)
		assert.Nil(t, err)
		assert.Equal(t, expected, acc.Balance())
	}

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

	acc, err = asBatch.GetAccount(acc1Address)
	assert.Nil(t, err)

	err = asBatch.AddBalance(acc1Address, amount)
	assert.Nil(t, err)

	acc, err = asBatch.GetAccount(acc1Address)
	assert.Equal(t, 2*amount, acc.Balance())
	checkBalance(acc1Address, amount)

	err = asBatch.Commit()
	assert.Nil(t, err)

	checkBalance(acc1Address, 2*amount)
}

func TestAccountStateBatch_Clone(t *testing.T) {
	s := getStorage(t)
	asBatch1, err := core.NewAccountStateBatch(nil, s)
	assert.Nil(t, err)

	asBatch2, err := asBatch1.Clone()
	assert.Nil(t, err)

	err = asBatch1.BeginBatch()
	assert.Nil(t, err)

	_, err = asBatch1.Clone()
	assert.Equal(t, core.ErrCannotCloneOnBatching, err)

	acc1Address := getAddress()
	amount := uint64(1)

	err = asBatch1.AddBalance(acc1Address, amount)
	assert.Nil(t, err)

	err = asBatch1.Commit()
	assert.Nil(t, err)

	_, err = asBatch1.AccountState().GetAccount(acc1Address)
	assert.Nil(t, err)
	_, err = asBatch2.AccountState().GetAccount(acc1Address)
	assert.NotNil(t, err)
}
