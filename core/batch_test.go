package core_test

import (
	"encoding/hex"
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/assert"
)

func testBatch(t *testing.T, batch *core.AccountStateBatch) {
	err := batch.BeginBatch()
	assert.Nil(t, err)

	err = batch.BeginBatch()
	assert.Equal(t, core.ErrBeginAgainInBatch, err)

	err = batch.RollBack()
	assert.Nil(t, err)

	err = batch.RollBack()
	assert.Equal(t, core.ErrNotBatching, err)
}

func getAddress() []byte {
	acc1Address, _ := hex.DecodeString("account1")
	return acc1Address
}

func getAmount() *util.Uint128 {
	amount, _ := util.NewUint128FromInt(1)
	return amount
}

func TestAccountState(t *testing.T) {
	s := testutil.GetStorage(t)

	asBatch, err := core.NewAccountStateBatch(nil, s)
	assert.Nil(t, err)
	assert.NotNil(t, asBatch)

	testBatch(t, asBatch)

	err = asBatch.BeginBatch()
	assert.Nil(t, err)

	acc1Address := getAddress()
	amount := getAmount()

	checkBalance := func(address []byte, expected *util.Uint128) {
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

	//_, err = asBatch.GetAccount(acc1Address)
	//assert.NotNil(t, err)

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
	two, _ := util.NewUint128FromInt(2)
	twoAmount, _ := amount.Mul(two)
	assert.Equal(t, twoAmount, acc.Balance())
	checkBalance(acc1Address, amount)

	err = asBatch.Commit()
	assert.Nil(t, err)

	checkBalance(acc1Address, twoAmount)
}

func TestAccountStateBatch_Clone(t *testing.T) {
	s := testutil.GetStorage(t)
	asBatch1, err := core.NewAccountStateBatch(nil, s)
	assert.Nil(t, err)

	asBatch2, err := asBatch1.Clone()
	assert.Nil(t, err)

	err = asBatch1.BeginBatch()
	assert.Nil(t, err)

	_, err = asBatch1.Clone()
	assert.Equal(t, core.ErrCannotCloneOnBatching, err)

	acc1Address := getAddress()
	amount := getAmount()

	err = asBatch1.AddBalance(acc1Address, amount)
	assert.Nil(t, err)

	err = asBatch1.Commit()
	assert.Nil(t, err)

	_, err = asBatch1.AccountState().GetAccount(acc1Address)
	assert.Nil(t, err)
	_, err = asBatch2.AccountState().GetAccount(acc1Address)
	assert.NotNil(t, err)
}
