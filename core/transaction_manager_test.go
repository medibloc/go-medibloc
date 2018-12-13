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
	"bytes"
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionManager_BroadcastAndRelay(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	node := testNetwork.NewNode()
	node.Start()
	testNetwork.WaitForEstablished()

	tb := blockutil.New(t, testutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx()

	tx := tb.RandomTx().Build()
	seed.Med.TransactionManager().Broadcast(tx)

	var actual *core.Transaction
	startTime := time.Now()
	for actual == nil || !bytes.Equal(tx.Hash(), actual.Hash()) {
		require.True(t, time.Now().Sub(startTime) < time.Duration(5*time.Second))
		actual = node.Med.TransactionManager().Pop()
		time.Sleep(10 * time.Millisecond)
	}

	tx = tb.RandomTx().Build()
	node.Med.TransactionManager().Relay(tx)

	actual = nil
	startTime = time.Now()
	for actual == nil || !bytes.Equal(tx.Hash(), actual.Hash()) {
		require.True(t, time.Now().Sub(startTime) < time.Duration(5*time.Second))
		actual = seed.Med.TransactionManager().Pop()
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTransactionManager_Push(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	tm := seed.Med.TransactionManager()

	randomTb := blockutil.New(t, testutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx().RandomTx()

	// Wrong chainID
	wrongChainIDTx := randomTb.ChainID(testutil.ChainID + 1).Build()
	assert.Equal(t, core.ErrInvalidChainID, tm.DisposeTx(wrongChainIDTx))

	// Wrong hash
	wrongHash := byteutils.Hex2Bytes("1234567890123456789012345678901234567890123456789012345678901234")
	wrongHashTx := randomTb.Hash(wrongHash).Build()
	assert.Equal(t, core.ErrInvalidTransactionHash, tm.DisposeTx(wrongHashTx))

	// No signature
	noSignTx := randomTb.Sign([]byte{}).Build()
	assert.Equal(t, core.ErrTransactionSignatureNotExist, tm.DisposeTx(noSignTx))

	//// Invalid signature
	//invalidSigner := testutil.NewAddrKeyPair(t)
	//invalidSignTx := randomTb.SignKey(invalidSigner.PrivKey).Build()
	//assert.Equal(t, core.ErrInvalidTransactionSigner, tm.Push(invalidSignTx))

	// No transactions on pool
	assert.Nil(t, tm.Pop())

	// Push duplicate transaction push
	tx := randomTb.Build()
	assert.NoError(t, tm.Push(tx))
	assert.Equal(t, core.ErrDuplicatedTransaction, tm.Push(tx))

}

func TestTransactionManager_PushAndRelay(t *testing.T) {
	numberOfNodes := 5

	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	seedTm := seed.Med.TransactionManager()

	for i := 1; i < numberOfNodes; i++ {
		testNetwork.NewNode().Start()
	}
	testNetwork.WaitForEstablished()

	randomTx := blockutil.New(t, testutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx().RandomTx().Build()

	require.NoError(t, seedTm.DisposeTx(randomTx))

	startTime := time.Now()
	relayCompleted := false
	for time.Now().Sub(startTime) < 10*time.Second && relayCompleted == false {
		relayCompleted = true
		for _, n := range testNetwork.Nodes {
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

func TestTransactionManager_GappedTransaction(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	tm := seed.Med.TransactionManager()

	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	from := bb.TokenDist[testutil.DynastySize]
	to := testutil.NewAddrKeyPair(t)

	tx1 := bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(10).Nonce(1).SignPair(from).Build()
	tx2 := bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(10).Nonce(2).SignPair(from).Build()
	tx3 := bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(10).Nonce(3).SignPair(from).Build()
	tx4 := bb.Tx().Type(core.TxOpTransfer).To(to.Addr).Value(10).Nonce(4).SignPair(from).Build()

	var err = *new(error)

	err = tm.DisposeTx(tx4)
	assert.NoError(t, err)
	assert.Nil(t, tm.Get(tx4.Hash()))
	assert.NotNil(t, tm.GetGappedTx(tx4.Hash()))

	err = tm.DisposeTx(tx3)
	assert.NoError(t, err)
	assert.Nil(t, tm.Get(tx3.Hash()))
	assert.NotNil(t, tm.GetGappedTx(tx3.Hash()))

	err = tm.DisposeTx(tx2)
	assert.NoError(t, err)
	assert.Nil(t, tm.Get(tx2.Hash()))
	assert.NotNil(t, tm.GetGappedTx(tx2.Hash()))

	err = tm.DisposeTx(tx1)
	assert.NoError(t, err)

	assert.NotNil(t, tm.Get(tx1.Hash()))
	assert.Nil(t, tm.GetGappedTx(tx1.Hash()))
	assert.NotNil(t, tm.Get(tx2.Hash()))
	assert.Nil(t, tm.GetGappedTx(tx2.Hash()))
	assert.NotNil(t, tm.Get(tx3.Hash()))
	assert.Nil(t, tm.GetGappedTx(tx3.Hash()))
	assert.NotNil(t, tm.Get(tx4.Hash()))
	assert.Nil(t, tm.GetGappedTx(tx4.Hash()))

}
