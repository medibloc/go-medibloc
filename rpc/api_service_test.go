package rpc_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"

	"github.com/gavv/httpexpect"

	"github.com/medibloc/go-medibloc/util/testutil"
)

func TestGetAccountApi(t *testing.T) {
	network := testutil.NewNetwork(t, 3)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist).Block(seed.GenesisBlock()).ChildWithTimestamp(dpos.
		NextMintSlot2(time.Now().Unix()))
	payer := seed.Config.TokenDist[0]

	tb := bb.Tx()
	tx1 := tb.Nonce(2).StakeTx(payer, 1000000000000000000).Build()
	b := bb.ExecuteTx(tx1).SignMiner().Build()

	seed.Med.BlockManager().PushBlockData(b.BlockData)

	e := httpexpect.New(t, testutil.IP2Local(seed.Config.Config.Rpc.HttpListen[0]))

	e.GET("/v1/account").
		WithQuery("address", payer.Addr).
		WithQuery("type", "tail").
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		ValueEqual("address", payer.Address()).
		ValueEqual("balance", "0").
		ValueEqual("nonce", "2").
		ValueEqual("vesting", "1000000000000000000").
		ValueEqual("voted", []string{}).
		ValueNotEqual("bandwidth", "0").
		ValueEqual("unstaking", "0")
}
