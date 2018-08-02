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

package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
)

func TestTransactionGetDel(t *testing.T) {
	tb := blockutil.New(t, testutil.DynastySize).Genesis().Child().Tx()

	tx1 := tb.RandomTx().Build()
	tx2 := tb.RandomTx().Build()

	pool := core.NewTransactionPool(128)
	pool.SetEventEmitter(core.NewEventEmitter(128))

	err := pool.Push(tx1)
	assert.NoError(t, err)
	err = pool.Push(tx2)
	assert.NoError(t, err)

	assert.Equal(t, tx1, pool.Get(tx1.Hash()))
	assert.Equal(t, tx2, pool.Get(tx2.Hash()))

	pool.Del(tx1)
	pool.Del(tx1)
	assert.Nil(t, pool.Get(tx1.Hash()))

	assert.Equal(t, tx2, pool.Get(tx2.Hash()))
	assert.Equal(t, tx2, pool.Pop())
	assert.Nil(t, pool.Get(tx2.Hash()))

	err = pool.Push(tx1)
	assert.Nil(t, err)
	err = pool.Push(tx2)
	assert.Nil(t, err)
	pool.Del(tx1)
	pool.Del(tx2)
	assert.Nil(t, pool.Pop())
}

func TestTransactionPoolEvict(t *testing.T) {
	var (
		nTransaction = 5
		poolSize     = 3
	)

	tb := blockutil.New(t, testutil.DynastySize).Genesis().Child().Tx()
	var txs []*core.Transaction
	for i := 0; i < nTransaction; i++ {
		tx := tb.RandomTx().Build()
		txs = append(txs, tx)
	}

	pool := core.NewTransactionPool(poolSize)
	pool.SetEventEmitter(core.NewEventEmitter(128))
	for _, tx := range txs {
		err := pool.Push(tx)
		assert.NoError(t, err)
	}

	for i := 3; i < nTransaction; i++ {
		assert.Nil(t, pool.Get(txs[i].Hash()))
	}
}

func TestEmptyPool(t *testing.T) {
	tb := blockutil.New(t, testutil.DynastySize).Genesis().Child().Tx()
	tx := tb.RandomTx().Build()

	pool := core.NewTransactionPool(128)
	assert.Nil(t, pool.Pop())
	assert.Nil(t, pool.Get(tx.Hash()))
	pool.Del(tx)
}
