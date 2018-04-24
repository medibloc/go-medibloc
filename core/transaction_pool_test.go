package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestTransactionPool(t *testing.T) {
	keys := test.NewKeySlice(t, 4)
	txs := []*core.Transaction{
		0: test.NewSignedTransaction(t, keys[0], keys[2], 0),
		1: test.NewSignedTransaction(t, keys[0], keys[2], 1),
		2: test.NewSignedTransaction(t, keys[2], keys[3], 2),
		3: test.NewSignedTransaction(t, keys[2], keys[1], 3),
		4: test.NewSignedTransaction(t, keys[0], keys[3], 0),
	}

	pool := core.NewTransactionPool(128)
	for _, tx := range txs {
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
	tx := test.NewRandomSignedTransaction(t)

	pool := core.NewTransactionPool(128)

	err := pool.Push(tx)
	assert.NoError(t, err)
	err = pool.Push(tx)
	assert.Equal(t, core.ErrDuplicatedTransaction, err)
}

func TestTransactionGetDel(t *testing.T) {
	tx1 := test.NewRandomSignedTransaction(t)
	tx2 := test.NewRandomSignedTransaction(t)

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
	keys := test.NewKeySlice(t, 4)
	txs := []*core.Transaction{
		0: test.NewSignedTransaction(t, keys[0], keys[1], 0),
		1: test.NewSignedTransaction(t, keys[1], keys[2], 1),
		2: test.NewSignedTransaction(t, keys[2], keys[3], 2),
		3: test.NewSignedTransaction(t, keys[3], keys[0], 3),
		4: test.NewSignedTransaction(t, keys[2], keys[1], 0),
	}

	pool := core.NewTransactionPool(3)
	for _, tx := range txs {
		err := pool.Push(tx)
		assert.NoError(t, err)
	}

	assert.Nil(t, pool.Get(txs[2].Hash()))
}

func TestEmptyPool(t *testing.T) {
	tx := test.NewRandomSignedTransaction(t)

	pool := core.NewTransactionPool(128)
	assert.Nil(t, pool.Pop())
	assert.Nil(t, pool.Get(tx.Hash()))
	pool.Del(tx)
}
