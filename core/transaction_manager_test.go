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
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionManager(t *testing.T) {
	mgrs, closeFn := testutil.NewTestTransactionManagers(t, 2)
	defer closeFn()

	tx := testutil.NewRandomSignedTransaction(t)

	mgrs[0].Broadcast(tx)
	var actual *core.Transaction
	for actual == nil {
		actual = mgrs[1].Pop()
		time.Sleep(time.Millisecond)
	}
	assert.EqualValues(t, tx.Hash(), actual.Hash())

	tx = testutil.NewRandomSignedTransaction(t)
	mgrs[1].Relay(tx)
	actual = nil
	for actual == nil {
		actual = mgrs[0].Pop()
		time.Sleep(time.Millisecond)
	}
	assert.EqualValues(t, tx.Hash(), actual.Hash())
}

func TestTransactionManagerAbnormalTx(t *testing.T) {
	mgrs, closeFn := testutil.NewTestTransactionManagers(t, 2)
	defer closeFn()

	sender, receiver := mgrs[0], mgrs[1]

	// No signature
	noSign := testutil.NewRandomTransaction(t)
	expectTxFiltered(t, sender, receiver, noSign)

	// Invalid signature
	from, to := testutil.NewPrivateKey(t), testutil.NewPrivateKey(t)
	invalidSign := testutil.NewTransaction(t, from, to, 10)
	testutil.SignTx(t, invalidSign, to)
	expectTxFiltered(t, sender, receiver, invalidSign)
}

func TestTransactionManagerDupTxFromNet(t *testing.T) {
	mgrs, closeFn := testutil.NewTestTransactionManagers(t, 2)
	defer closeFn()

	sender, receiver := mgrs[0], mgrs[1]

	dup := testutil.NewRandomSignedTransaction(t)
	sender.Broadcast(dup)
	sender.Broadcast(dup)
	time.Sleep(100 * time.Millisecond)

	normal := testutil.NewRandomSignedTransaction(t)
	sender.Broadcast(normal)

	var count int
	for {
		recv := receiver.Pop()
		if recv != nil && byteutils.Equal(recv.Hash(), normal.Hash()) {
			break
		}
		if recv != nil {
			count++
		}
		time.Sleep(time.Millisecond)
	}
	assert.Equal(t, 1, count)
}

func TestTransactionManagerDupTxPush(t *testing.T) {
	mgrs, closeFn := testutil.NewTestTransactionManagers(t, 2)
	defer closeFn()

	dup := testutil.NewRandomSignedTransaction(t)
	err := mgrs[0].Push(dup)
	assert.NoError(t, err)
	err = mgrs[0].Push(dup)
	assert.EqualValues(t, core.ErrDuplicatedTransaction, err)

	actual := mgrs[0].Pop()
	assert.EqualValues(t, dup, actual)
	actual = mgrs[0].Pop()
	assert.Nil(t, actual)
}

func TestTransactionManager_PushAndRelay(t *testing.T) {
	dynastySize := testutil.DynastySize

	tn := testutil.NewNetwork(t, dynastySize)
	defer tn.Cleanup()
	tn.SetLogTestHook()
	seed := tn.NewSeedNode()
	seed.Start()

	for i := 1; i < dynastySize; i++ {
		tn.NewNode().Start()
	}
	tn.WaitForEstablished()

	bb := blockutil.New(t, dynastySize).Block(seed.Tail()).Child()
	tx := bb.
		Tx().Type(core.TxOpTransfer).To(seed.Config.TokenDist[1].Addr).Value(10).SignPair(seed.Config.TokenDist[0]).Build()

	require.NoError(t, seed.Med.TransactionManager().PushAndRelay(tx))

	startTime := time.Now()
	relayCompleted := false
	for time.Now().Sub(startTime) < 10*time.Second && relayCompleted == false {
		relayCompleted = true
		for _, n := range tn.Nodes {
			txs := n.Med.TransactionManager().GetAll()
			if len(txs) == 0 {
				relayCompleted = false
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("Waiting time to relay tx: %v", time.Now().Sub(startTime))
	assert.True(t, relayCompleted)
}

func expectTxFiltered(t *testing.T, sender, receiver *core.TransactionManager, abnormal *core.Transaction) {
	sender.Broadcast(abnormal)

	time.Sleep(100 * time.Millisecond)

	normal := testutil.NewRandomSignedTransaction(t)
	sender.Broadcast(normal)

	var recv *core.Transaction
	for recv == nil {
		recv = receiver.Pop()
		time.Sleep(time.Millisecond)
	}
	assert.EqualValues(t, normal.Hash(), recv.Hash())
}
