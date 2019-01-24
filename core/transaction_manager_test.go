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
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionManager_Broadcast(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	node := testNetwork.NewNode()
	node.Start()
	testNetwork.WaitForEstablished()

	tb := blockutil.New(t, blockutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx()

	tx := tb.RandomTx().Build()
	require.NoError(t, seed.Med.TransactionManager().Broadcast(tx))

	var actual *transaction.ExecutableTx
	startTime := time.Now()
	for actual == nil || !bytes.Equal(tx.Hash(), actual.Hash()) {
		require.True(t, time.Now().Sub(startTime) < time.Duration(5*time.Second))
		actual = node.Med.TransactionManager().Pop()
		time.Sleep(10 * time.Millisecond)
	}

	tx = tb.RandomTx().Build()
	require.NoError(t, node.Med.TransactionManager().Broadcast(tx))

	actual = nil
	startTime = time.Now()
	for actual == nil || !bytes.Equal(tx.Hash(), actual.Hash()) {
		require.True(t, time.Now().Sub(startTime) < time.Duration(5*time.Second))
		actual = seed.Med.TransactionManager().Pop()
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTransactionManager_Push(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	tm := seed.Med.TransactionManager()

	randomTb := blockutil.New(t, blockutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx().RandomTx()

	// Wrong chainID
	wrongChainIDTx := randomTb.ChainID(blockutil.ChainID + 1).Build()
	failed := tm.PushAndExclusiveBroadcast(wrongChainIDTx)
	assert.Equal(t, ErrInvalidTxChainID, failed[wrongChainIDTx.HexHash()])

	// Wrong hash
	wrongHash, err := byteutils.Hex2Bytes("1234567890123456789012345678901234567890123456789012345678901234")
	require.NoError(t, err)
	wrongHashTx := randomTb.Hash(wrongHash).Build()
	failed = tm.PushAndExclusiveBroadcast(wrongHashTx)
	assert.Equal(t, ErrInvalidTransactionHash, failed[wrongHashTx.HexHash()])

	// No signature
	noSignTx := randomTb.Sign([]byte{}).Build()
	failed = tm.PushAndExclusiveBroadcast(noSignTx)
	assert.Equal(t, ErrTransactionSignatureNotExist, failed[noSignTx.HexHash()])

	// // Invalid signature
	// invalidSigner := testutil.NewAddrKeyPair(t)
	// invalidSignTx := randomTb.SignKey(invalidSigner.PrivKey).Build()
	// assert.Equal(t, core.ErrInvalidTransactionSigner, tm.Push(invalidSignTx))

	// No transactions on pool
	assert.Nil(t, tm.Pop())

	// Push duplicate transaction push
	tx := randomTb.Build()
	failed = tm.PushAndExclusiveBroadcast(tx)
	assert.Nil(t, failed[tx.HexHash()])
	failed = tm.PushAndExclusiveBroadcast(tx)
	assert.Equal(t, core.ErrDuplicatedTransaction, failed[tx.HexHash()])

}

func TestTransactionManager_PushAndBroadcast(t *testing.T) {
	const (
		timeout       = 50 * time.Millisecond
		numberOfNodes = 3
	)
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	seedTm := seed.Med.TransactionManager()

	for i := 1; i < numberOfNodes; i++ {
		testNetwork.NewNode().Start()
	}
	testNetwork.WaitForEstablished()

	tb := blockutil.New(t, blockutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx()
	randomTx1 := tb.RandomTx().Build()
	randomTx2 := tb.Nonce(randomTx1.Nonce() + 1).RandomTx().Build()

	failed := seedTm.PushAndBroadcast(randomTx2)
	require.Nil(t, failed[randomTx2.HexHash()])

	for i, n := range testNetwork.Nodes {
		if i == 0 {
			continue
		}
		waitForTransaction(t, timeout, n, randomTx2, false)
	}

	failed = seedTm.PushAndBroadcast(randomTx1)
	require.Nil(t, failed[randomTx1.HexHash()])

	startTime := time.Now()
	for _, n := range testNetwork.Nodes {
		waitForTransaction(t, timeout, n, randomTx1, true)
		waitForTransaction(t, timeout, n, randomTx2, true)
	}

	t.Logf("Waiting time to broadcast tx: %v", time.Now().Sub(startTime))
}

func TestTransactionManager_PushAndExclusiveBroadcast(t *testing.T) {
	const (
		timeout       = 50 * time.Millisecond
		numberOfNodes = 3
	)

	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	seedTm := seed.Med.TransactionManager()

	for i := 1; i < numberOfNodes; i++ {
		testNetwork.NewNode().Start()
	}
	testNetwork.WaitForEstablished()

	tb := blockutil.New(t, blockutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx()
	randomTx1 := tb.RandomTx().Build()
	randomTx2 := tb.Nonce(randomTx1.Nonce() + 1).RandomTx().Build()

	failed := seedTm.PushAndBroadcast(randomTx2)
	require.Nil(t, failed[randomTx2.HexHash()])

	for i, n := range testNetwork.Nodes {
		if i == 0 {
			continue
		}
		waitForTransaction(t, timeout, n, randomTx2, false)
	}

	failed = seedTm.PushAndExclusiveBroadcast(randomTx1)
	require.Nil(t, failed[randomTx1.HexHash()])

	startTime := time.Now()
	for i, n := range testNetwork.Nodes {
		if i == 0 {
			waitForTransaction(t, timeout, n, randomTx1, true)
		}
		waitForTransaction(t, timeout, n, randomTx2, true)
	}
	t.Logf("Waiting time to broadcast tx: %v", time.Now().Sub(startTime))
}

func TestTransactionManager_MaxPending(t *testing.T) {
	const (
		timeout              = 50 * time.Millisecond
		numberOfNodes        = 3
		numberOfTransactions = 100
		numberOfPop          = 10
		numberOfDel          = 5
	)
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	seedTm := seed.Med.TransactionManager()

	for i := 1; i < numberOfNodes; i++ {
		testNetwork.NewNode().Start()
	}
	testNetwork.WaitForEstablished()

	txs := blockutil.New(t, blockutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).
		Tx().RandomTxs(numberOfTransactions)

	failed := seedTm.PushAndBroadcast(txs...)
	require.Zero(t, len(failed))

	for i, n := range testNetwork.Nodes {
		for j := 0; j < core.MaxPending; j++ {
			waitForTransaction(t, timeout, n, txs[j], true)
		}
		for j := core.MaxPending; j < numberOfTransactions; j++ {
			waitForTransaction(t, timeout, n, txs[j], i == 0)
		}
	}

	for i := 0; i < numberOfPop; i++ {
		seedTm.Pop()
	}

	for _, n := range testNetwork.Nodes {
		for j := core.MaxPending; j < core.MaxPending+numberOfPop; j++ {
			waitForTransaction(t, timeout, n, txs[j], true)
		}
	}

	addr := txs[numberOfPop+numberOfDel-1].From()
	nonce := txs[numberOfPop+numberOfDel-1].Nonce()

	seedTm.DelByAddressNonce(addr, nonce)

	for _, n := range testNetwork.Nodes {
		for j := core.MaxPending + numberOfPop; j < core.MaxPending+numberOfPop+numberOfDel; j++ {
			waitForTransaction(t, timeout, n, txs[j], true)
		}
	}

	// seedTm.PushAndExclusiveBroadcast(txs[numberOfPop:numberOfPop+numberOfDel]...)
}

func TestTransactionManager_BandwidthLimit(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	seedTm := seed.Med.TransactionManager()

	from := seed.Config.TokenDist[blockutil.DynastySize]

	bb := blockutil.New(t, blockutil.DynastySize).AddKeyPairs(seed.Config.Dynasties).Block(seed.Tail()).Child().SignProposer()
	err := seed.Med.BlockManager().PushBlockDataSync(bb.Build().GetBlockData(), 1000*time.Millisecond)
	assert.NoError(t, err)

	tx1 := bb.Tx().Type(coreState.TxOpTransfer).Value(1000).SignPair(from).Build()
	tx2 := bb.Tx().Type(coreState.TxOpStake).Value(1000).SignPair(from).Build()
	tx3 := bb.Tx().Type(coreState.TxOpUnstake).Nonce(tx2.Nonce() + 1).Value(1000).SignPair(from).Build()

	failed := seedTm.PushAndExclusiveBroadcast(tx1)
	err = failed[byteutils.Bytes2Hex(tx1.Hash())]
	require.Equal(t, ErrPointNotEnough, err)

	failed = seedTm.PushAndExclusiveBroadcast(tx2)
	err = failed[byteutils.Bytes2Hex(tx2.Hash())]
	require.Nil(t, err)

	failed = seedTm.PushAndExclusiveBroadcast(tx3)
	err = failed[byteutils.Bytes2Hex(tx3.Hash())]
	require.Equal(t, ErrStakingNotEnough, err)

}

func TestTransactionManager_ReplacePending(t *testing.T) {
	const (
		numberOfTransactions = 10
		newReplaceAllowTime  = 1 * time.Second
	)
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	seedTm := seed.Med.TransactionManager()

	tb := blockutil.New(t, blockutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist).Tx()

	txs := tb.RandomTxs(numberOfTransactions)

	failed := seedTm.PushAndBroadcast(txs...)
	require.Zero(t, len(failed))

	newTxs := tb.RandomTxs(numberOfTransactions)
	failed = seedTm.PushAndBroadcast(newTxs...)
	require.Equal(t, numberOfTransactions, len(failed))
	for _, err := range failed {
		require.Equal(t, err, core.ErrFailedToReplacePendingTx)
	}

	core.AllowReplacePendingDuration = newReplaceAllowTime
	time.Sleep(newReplaceAllowTime)

	failed = seedTm.PushAndBroadcast(newTxs...)
	require.Zero(t, len(failed))

	for _, tx := range txs {
		require.Nil(t, seedTm.Get(tx.Hash()))
	}
}

func waitForTransaction(t *testing.T, deadline time.Duration, node *testutil.Node, tx *Transaction, has bool) {
	timeout := time.NewTimer(deadline)
	defer timeout.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout.C:
			require.False(t, has, "failed to wait transaction")
			return
		case <-ticker.C:
			receivedTx := node.Med.TransactionManager().Get(tx.Hash())
			if receivedTx != nil {
				require.True(t, has)
				return
			}
		}
	}
}
