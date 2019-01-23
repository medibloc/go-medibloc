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
	"time"

	"github.com/medibloc/go-medibloc/core"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/event"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionGetDel(t *testing.T) {
	tb := blockutil.New(t, testutil.DynastySize).Genesis().Child().Tx()

	tx1 := tb.RandomTx().Build()
	tx2 := tb.Nonce(tx1.Nonce() + 1).RandomTx().Build()

	pool := core.NewTransactionPool(128)
	pool.SetEventEmitter(event.NewEventEmitter(128))

	tp1, err := core.NewTransactionContext(tx1)
	require.NoError(t, err)
	tp2, err := core.NewTransactionContext(tx2)
	require.NoError(t, err)

	pool.Push(tp1)
	pool.Push(tp2)

	assert.Equal(t, tx1, pool.Get(tx1.Hash()).Transaction)
	assert.Equal(t, tx2, pool.Get(tx2.Hash()).Transaction)

	pool.Del(tx1.Hash())
	assert.Nil(t, pool.Get(tx1.Hash()))

	assert.Equal(t, tx2, pool.Get(tx2.Hash()).Transaction)
	assert.Equal(t, tx2, pool.Pop().Transaction)
	assert.Nil(t, pool.Get(tx2.Hash()))

	pool.Push(tp1)
	pool.Push(tp2)

	pool.Del(tp1.Hash())
	pool.Del(tp2.Hash())
	assert.Nil(t, pool.Pop())
}

func TestTransactionPoolEvict(t *testing.T) {
	var (
		nTransaction = 5
		poolSize     = 3
	)

	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()
	from := bb.KeyPairs[0]
	to := keyutil.NewAddrKeyPair(t)

	tb := bb.Tx()
	var txs []*coreState.Transaction
	for i := 0; i < nTransaction; i++ {
		tx := tb.Type(coreState.TxOpTransfer).Value(10).To(to.Addr).Nonce(uint64(i + 1)).SignPair(from).Build()
		txs = append(txs, tx)
	}

	pool := core.NewTransactionPool(poolSize)
	pool.SetEventEmitter(event.NewEventEmitter(128))
	for _, tx := range txs {
		tp, err := core.NewTransactionContext(tx)
		require.NoError(t, err)
		pool.Push(tp)
	}
	pool.Evict()
	assert.Equal(t, 3, len(pool.GetAll()))
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
	pool.Del(tx.Hash())
}

func TestInfiniteLoop(t *testing.T) {
	tb := blockutil.New(t, testutil.DynastySize).Genesis().Child().Tx()
	from1 := keyutil.NewAddrKeyPair(t)
	from2 := keyutil.NewAddrKeyPair(t)
	to := keyutil.NewAddrKeyPair(t)

	from1Nonce2 := tb.Type(coreState.TxOpTransfer).Value(10).To(to.Addr).Nonce(2).
		SignPair(from1).Build()
	from2Nonce1 := tb.Type(coreState.TxOpTransfer).Value(10).To(to.Addr).Nonce(1).
		SignPair(from2).Build()
	from2Nonce2 := tb.Type(coreState.TxOpTransfer).Value(10).To(to.Addr).Nonce(2).
		SignPair(from2).Build()

	tmp1, err := core.NewTransactionContext(from1Nonce2)
	require.NoError(t, err)
	tmp2, err := core.NewTransactionContext(from2Nonce1)
	require.NoError(t, err)
	tmp2.SetIncomeTimestamp(tmp1.IncomeTimestamp() + 1000)
	tmp3, err := core.NewTransactionContext(from2Nonce2)
	require.NoError(t, err)
	tmp3.SetIncomeTimestamp(tmp2.IncomeTimestamp() + 1000)

	pool := core.NewTransactionPool(128)
	pool.Push(tmp1)
	require.NoError(t, err)
	pool.Push(tmp2)
	require.NoError(t, err)
	pool.Push(tmp3)
	require.NoError(t, err)

	tx := pool.Pop()
	require.Equal(t, from1.Addr.Hex(), tx.From().Hex())
	require.Equal(t, uint64(2), tx.Nonce())
	time.Sleep(time.Millisecond)
	tx.SetIncomeTimestamp(time.Now().UnixNano())
	pool.Push(tx)

	tx = pool.Pop()
	require.Equal(t, from2.Addr, tx.From())
	require.Equal(t, uint64(1), tx.Nonce())
	time.Sleep(time.Millisecond)
	tx.SetIncomeTimestamp(time.Now().UnixNano())
	pool.Push(tx)

	tx = pool.Pop()
	require.Equal(t, from1.Addr, tx.From())
	require.Equal(t, uint64(2), tx.Nonce())
	time.Sleep(time.Millisecond)
	tx.SetIncomeTimestamp(time.Now().UnixNano())
	pool.Push(tx)

	tx = pool.Pop()
	require.Equal(t, from2.Addr, tx.From())
	require.Equal(t, uint64(1), tx.Nonce())

	tx = pool.Pop()
	require.Equal(t, from2.Addr, tx.From())
	require.Equal(t, uint64(2), tx.Nonce())

	tx = pool.Pop()
	require.Equal(t, from1.Addr, tx.From())
	require.Equal(t, uint64(2), tx.Nonce())

	tx = pool.Pop()
	require.Nil(t, tx)
}
