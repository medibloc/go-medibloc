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

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CASE 1. Receipt store and get from storage
// CASE 2. Get transaction from another node and check receipt
// CASE 3. Make invalid transaction and receipt should hold matched error message

func TestReceipt(t *testing.T) {
	network := testutil.NewNetwork(t, testutil.DynastySize)
	defer network.Cleanup()
	network.SetLogTestHook()

	seed := network.NewSeedNode()
	seed.Start()
	receiver := network.NewNode()
	receiver.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist).Block(seed.GenesisBlock()).ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix()))
	payer := seed.Config.TokenDist[testutil.DynastySize]

	tb := bb.Tx()
	tx1 := tb.Nonce(1).StakeTx(payer, 100).Build()
	tx2 := tb.Nonce(2).Type(core.TxOpWithdrawVesting).Value(200).SignPair(payer).
		Build()
	tx3 := tb.Nonce(2).Type(core.TxOpWithdrawVesting).Value(50).SignPair(payer).Build()
	b := bb.ExecuteTx(tx1).ExecuteTxErr(tx2, core.ErrVestingNotEnough).ExecuteTx(tx3).SignMiner().Build()

	seed.Med.BlockManager().PushBlockData(b.BlockData)
	seed.Med.BlockManager().BroadCast(b.BlockData)

	time.Sleep(1000 * time.Millisecond)
	tx1r, err := receiver.Tail().State().GetTx(tx1.Hash())
	assert.NoError(t, err)
	assert.True(t, tx1r.Receipt().Executed())
	assert.Equal(t, tx1r.Receipt().Error(), []byte(nil))

	tx2r, err := receiver.Tail().State().GetTx(tx2.Hash())
	assert.Equal(t, err, core.ErrNotFound)
	assert.Nil(t, tx2r)

	tx3r, err := receiver.Tail().State().GetTx(tx3.Hash())
	assert.NoError(t, err)
	assert.True(t, tx3r.Receipt().Executed())
	assert.Equal(t, tx3r.Receipt().Error(), []byte(nil))
}

func TestErrorTransactionReceipt(t *testing.T) {
	network := testutil.NewNetwork(t, testutil.DynastySize)
	defer network.Cleanup()
	network.SetLogTestHook()

	seed := network.NewSeedNode()
	seed.Start()
	receiver := network.NewNode()
	receiver.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, testutil.DynastySize).AddKeyPairs(seed.Config.TokenDist).
		Block(seed.GenesisBlock()).ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix()))
	payer := seed.Config.TokenDist[testutil.DynastySize]

	tb := bb.Tx()
	tx1 := tb.Nonce(1).StakeTx(payer, 1000).Build()

	payload := &core.AddRecordPayload{
		RecordHash: byteutils.Hex2Bytes("9eca7128409f609b2a72fc24985645665bbb99152b4b14261c3c3c93fb17cf54"),
	}

	tx2 := tb.Nonce(2).Type(core.TxOpAddRecord).Payload(payload).SignPair(payer).Build()
	tx3 := tb.Nonce(3).Type(core.TxOpAddRecord).Payload(payload).SignPair(payer).Build()
	b := bb.ExecuteTx(tx1).ExecuteTx(tx2).ExecuteTxErr(tx3, core.ErrRecordAlreadyAdded).SignMiner().Build()

	seed.Med.BlockManager().PushBlockData(b.BlockData)
	seed.Med.BlockManager().BroadCast(b.BlockData)

	time.Sleep(1000 * time.Millisecond)
	tx1r, err := receiver.Tail().State().GetTx(tx1.Hash())
	assert.NoError(t, err)
	assert.True(t, tx1r.Receipt().Executed())
	assert.Equal(t, tx1r.Receipt().Error(), []byte(nil))

	tx2r, err := receiver.Tail().State().GetTx(tx2.Hash())
	assert.NoError(t, err)
	assert.True(t, tx2r.Receipt().Executed())
	assert.Equal(t, tx2r.Receipt().Error(), []byte(nil))

	tx3r, err := receiver.Tail().State().GetTx(tx3.Hash())
	assert.NoError(t, err)
	assert.False(t, tx3r.Receipt().Executed())
	assert.Equal(t, string(tx3r.Receipt().Error()[:]), core.ErrRecordAlreadyAdded.Error())
}

func TestWrongReceipt(t *testing.T) {
	network := testutil.NewNetwork(t, testutil.DynastySize)
	defer network.Cleanup()
	network.SetLogTestHook()

	seed := network.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, testutil.DynastySize).AddKeyPairs(seed.Config.TokenDist).
		Block(seed.GenesisBlock()).Child()

	b := bb.Tx().RandomTx().Execute().SignMiner().Build()

	b.Transactions()[0].Receipt().SetCPUUsage(util.NewUint128())

	require.Equal(t, core.ErrCannotExecuteOnParentBlock, seed.Med.BlockManager().PushBlockData(b.BlockData))

}
