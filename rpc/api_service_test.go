package rpc_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/util/byteutils"

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

func TestGetBlockApi(t *testing.T) {
	network := testutil.NewNetwork(t, 3)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist)
	b := bb.Block(seed.GenesisBlock()).
		ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix())).
		Stake().Tx().RandomTx().Execute().SignMiner().Build()

	seed.Med.BlockManager().PushBlockData(b.BlockData)

	e := httpexpect.New(t, testutil.IP2Local(seed.Config.Config.Rpc.HttpListen[0]))

	// The block response should be the same one for designated hash, type, height parameter
	e.GET("/v1/block").
		WithQuery("hash", byteutils.Bytes2Hex(b.Hash())).
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		ValueEqual("height", "2")

	e.GET("/v1/block").
		WithQuery("type", "tail").
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		ValueEqual("height", "2")

	e.GET("/v1/block").
		WithQuery("height", "2").
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		ValueEqual("height", "2")

	// The block response should be the genesis one
	e.GET("/v1/block").
		WithQuery("type", "genesis").
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		ValueEqual("height", "1")

	// Check block response parameters
	e.GET("/v1/block").
		WithQuery("type", "tail").
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		ContainsKey("height").
		ContainsKey("hash").
		ContainsKey("parent_hash").
		ContainsKey("coinbase").
		ContainsKey("reward").
		ContainsKey("supply").
		ContainsKey("timestamp").
		ContainsKey("chain_id").
		ContainsKey("alg").
		ContainsKey("sign").
		ContainsKey("accs_root").
		ContainsKey("txs_root").
		ContainsKey("dpos_root").
		ContainsKey("transactions")
}

func TestGetBlocksApi(t *testing.T) {
	network := testutil.NewNetwork(t, 3)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	network.WaitForEstablished()

	bb := blockutil.New(t, 3).AddKeyPairs(seed.Config.TokenDist)
	b := bb.Block(seed.GenesisBlock()).
		ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix())).SignMiner().Build()

	seed.Med.BlockManager().PushBlockData(b.BlockData)

	e := httpexpect.New(t, testutil.IP2Local(seed.Config.Config.Rpc.HttpListen[0]))

	schema := `
	{
		"type":"object",
		"properties":{
			"blocks":{
				"type":"array",
				"items":{
					"type":"object",
					"properties":{
						"height": {
							"type":"string"
						}
					}
				},
				"minItems":2
			}
		}
	}`

	e.GET("/v1/blocks").
		WithQuery("from", "1").
		WithQuery("to", "2").
		Expect().JSON().Schema(schema)
}

func TestGetCandidatesApi(t *testing.T) {
	network := testutil.NewNetwork(t, 3)
	defer network.Cleanup()

	seed := network.NewSeedNode()
	seed.Start()
	network.WaitForEstablished()

	e := httpexpect.New(t, testutil.IP2Local(seed.Config.Config.Rpc.HttpListen[0]))

	e.GET("/v1/candidates").
		Expect().JSON().
		Path("$.candidates").
		Array().Length().Equal(3)
}
