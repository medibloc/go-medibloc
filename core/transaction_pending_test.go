package core_test

import (
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/medibloc/go-medibloc/util/testutil/retry"
	"github.com/stretchr/testify/require"
)

func TestPendingTransactionPool_TransferAfterStake(t *testing.T) {
	tn := testutil.NewNetwork(t)
	defer tn.Cleanup()

	seed := tn.NewSeedNode()
	tn.AddProposerFromDynasties(seed)
	seed.Start()

	bb := blockutil.New(t, tn.DynastySize).Block(seed.GenesisBlock()).AddKeyPairs(seed.Config.TokenDist)

	from := seed.Config.TokenDist[len(seed.Config.TokenDist)-1]
	nonce := bb.Expect().GetNonce(from.Addr)

	to := keyutil.NewAddrKeyPair(t)

	stakeTx := bb.Tx().Type(transaction.TxOpStake).Value(10000000).SignPairWithNonce(from, nonce+1).Build()
	transferTx := bb.Tx().Type(transaction.TxOpTransfer).Value(10000000).To(to.Addr).SignPairWithNonce(from, nonce+2).Build()

	tm := seed.Med.TransactionManager()

	err := tm.Push(stakeTx)
	require.NoError(t, err)
	err = tm.Push(transferTx)
	require.NoError(t, err)

	r := retry.New(t, 10, time.Second)
	r.Try(func(t retry.T) { require.Equal(t, 0, tm.Len()) })
}

// func mockAccount(addr common.Address, nonce uint64, staking uint64, points uint64, lastPointsTs time.Time) *corestate.Account {
// 	return &core.Account{
// 		Address:      addr,
// 		Nonce:        nonce,
// 		Staking:      util.NewUint128FromUint(staking),
// 		Points:       util.NewUint128FromUint(points),
// 		LastPointsTs: lastPointsTs.Unix(),
// 	}
// }
//
// func mockPrice(cpu, net uint64) core.Price {
// 	p := core.Price{}
// 	p.SetCpuPrice(util.NewUint128FromUint(cpu))
// 	p.SetNetPrice(util.NewUint128FromUint(net))
// 	return p
// }
//
// func TestPendingTransactionPool_PushOrReplace(t *testing.T) {
// 	pool := core.NewPendingTransactionPool()
//
// 	from := testutil.NewAddrKeyPair(t)
// 	to := testutil.NewAddrKeyPair(t)
//
// 	acc := mockAccount(from.Addr, 0, 1000, 1000, time.Now())
// 	price := mockPrice(1, 1)
//
// 	bb := blockutil.New(t, testutil.DynastySize)
// 	ttx := bb.Tx().Type(core.TxOpTransfer).From(from.Addr).Nonce(1).To(to.Addr).Build()
// 	tx, err := core.NewTxContext(ttx)
//
// 	err = pool.PushOrReplace(tx, acc, price)
// 	require.NoError(t, err)
// }

/*

//RandomTx generate random Tx
func (tb *TxBuilder) RandomTx() *TxBuilder {
	n := tb.copy()
	require.NotEqual(n.t, 0, len(n.bb.KeyPairs), "No key pair added on block builder")

	from := n.bb.KeyPairs[0]
	to := testutil.NewAddrKeyPair(n.t)
	return n.Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from)
}

*/
