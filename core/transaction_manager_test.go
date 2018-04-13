package core_test

import (
	"testing"

	"time"

	"github.com/medibloc/go-medibloc/core"
	"github.com/stretchr/testify/assert"
)

func TestTransactionManager(t *testing.T) {
	mgrs, closeFn := newTestTransactionManagers(t, 2)
	defer closeFn()

	tx := newRandomSignedTransaction(t)

	mgrs[0].Broadcast(tx)
	var actual *core.Transaction
	for actual == nil {
		actual = mgrs[1].Pop()
		time.Sleep(time.Millisecond)
	}
	assert.EqualValues(t, tx.Hash(), actual.Hash())

	tx = newRandomSignedTransaction(t)
	mgrs[1].Relay(tx)
	actual = nil
	for actual == nil {
		actual = mgrs[0].Pop()
		time.Sleep(time.Millisecond)
	}
	assert.EqualValues(t, tx.Hash(), actual.Hash())
}
