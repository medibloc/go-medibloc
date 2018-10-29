package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/medibloc/go-medibloc/core"

	"github.com/medibloc/go-medibloc/consensus/dpos"

	"github.com/medibloc/go-medibloc/util/testutil/blockutil"

	"github.com/medibloc/go-medibloc/util/testutil"
)

// CASE 1. Receipt store and get from storage
// CASE 2. Get transaction from another node and check receipt
// CASE 3. Make invalid transaction and receipt should hold matched error message

func TestReceipt(t *testing.T) {
	network := testutil.NewNetwork(t, 3)
	//defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	receiver := network.NewNode()
	receiver.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist).Block(seed.GenesisBlock()).ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix()))
	payer := seed.Config.TokenDist[0]

	tb := bb.Tx()
	tx1 := tb.Nonce(2).StakeTx(payer, 1000000000000000).Build()
	tx2 := tb.Nonce(3).From(payer.Addr).Type(core.TxOpWithdrawVesting).Value(2000000000000000).SignPair(payer).Build()
	tx3 := tb.Nonce(3).From(payer.Addr).Type(core.TxOpWithdrawVesting).Value(90000000000000).SignPair(payer).Build()
	b := bb.ExecuteTx(tx1).ExecuteTxErr(tx2, core.ErrExecutedErr).ExecuteTx(tx3).SignMiner().Build()

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
