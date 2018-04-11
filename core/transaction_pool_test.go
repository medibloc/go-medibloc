package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/stretchr/testify/assert"
)

func TestTransactionPool(t *testing.T) {
	addr := newTestAddrSet(t, 4)
	txs := map[int]*core.Transaction{
		0: newTestTransaction(t, addr[0], addr[2], 0),
		1: newTestTransaction(t, addr[0], addr[2], 1),
		2: newTestTransaction(t, addr[2], addr[3], 2),
		3: newTestTransaction(t, addr[2], addr[1], 3),
		4: newTestTransaction(t, addr[0], addr[3], 0),
	}

	pool := core.NewTransactionPool(128)
	for _, tx := range txs {
		signTx(t, tx)
		err := pool.Push(tx)
		assert.NoError(t, err)
	}

	tx := pool.Pop()
	assert.True(t, txs[0] == tx || txs[4] == tx)
	tx = pool.Pop()
	assert.True(t, txs[0] == tx || txs[4] == tx)
	assert.Equal(t, txs[1], pool.Pop())
	assert.Equal(t, txs[2], pool.Pop())
	assert.Equal(t, txs[3], pool.Pop())
	assert.Nil(t, pool.Pop())
}

func TestDuplicatedTx(t *testing.T) {
	addr := RandomAddress(t)
	tx := newTestTransaction(t, addr, addr, 0)
	signTx(t, tx)

	pool := core.NewTransactionPool(128)

	err := pool.Push(tx)
	assert.NoError(t, err)
	err = pool.Push(tx)
	assert.Equal(t, core.ErrDuplicatedTransaction, err)
}

func TestTransactionGetDel(t *testing.T) {
	from, to := RandomAddress(t), RandomAddress(t)
	tx1 := newTestTransaction(t, from, to, 5)
	signTx(t, tx1)
	tx2 := newTestTransaction(t, from, to, 10)
	signTx(t, tx2)

	pool := core.NewTransactionPool(128)
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
	addr := newTestAddrSet(t, 4)
	txs := map[int]*core.Transaction{
		0: newTestTransaction(t, addr[0], addr[1], 0),
		1: newTestTransaction(t, addr[1], addr[2], 1),
		2: newTestTransaction(t, addr[2], addr[3], 2),
		3: newTestTransaction(t, addr[3], addr[0], 3),
		4: newTestTransaction(t, addr[2], addr[1], 0),
	}

	pool := core.NewTransactionPool(3)
	for _, tx := range txs {
		signTx(t, tx)
		err := pool.Push(tx)
		assert.NoError(t, err)
	}

	assert.Nil(t, pool.Get(txs[2].Hash()))
}

func TestEmptyPool(t *testing.T) {
	from, to := RandomAddress(t), RandomAddress(t)
	tx := newTestTransaction(t, from, to, 10)

	pool := core.NewTransactionPool(128)
	assert.Nil(t, pool.Pop())
	assert.Nil(t, pool.Get(tx.Hash()))
	pool.Del(tx)
}
