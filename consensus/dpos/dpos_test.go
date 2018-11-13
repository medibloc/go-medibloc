package dpos_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeNewDynasty(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()
	b := bb.Build()

	as := b.State().AccState()
	ds := b.State().DposState()

	genesisDynasty, err := ds.Dynasty()
	require.NoError(t, err)
	t.Log("genesis dynasty", genesisDynasty)

	candidates, err := ds.SortByVotePower(as)
	require.NoError(t, err)

	newDynasty := dpos.MakeNewDynasty(candidates, testutil.DynastySize)
	t.Log("new dynasty:", newDynasty)
}

func TestChangeDynasty(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	newCandidate := seed.Config.TokenDist[testutil.DynastySize]

	bb := blockutil.New(t, testutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist)

	t.Log(seed.Tail().Hash())
	t.Log(bb.B.Hash())

	bb = bb.Child().SignProposer()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))

	bb = bb.Child().Stake().
		Tx().Type(dpos.TxOpBecomeCandidate).Value(0).SignPair(newCandidate).Execute().
		Tx().Type(core.TxOpVest).Value(10).SignPair(newCandidate).Execute().
		Tx().Type(dpos.TxOpVote).
		Payload(&dpos.VotePayload{
			Candidates: []common.Address{newCandidate.Addr},
		}).SignPair(newCandidate).Execute().SignProposer()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	ds := seed.Tail().State().DposState()
	isCandidate, err := ds.IsCandidate(newCandidate.Addr)

	require.NoError(t, err)
	assert.True(t, isCandidate)
	inDynasty, err := ds.InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.False(t, inDynasty)

	bb = bb.ChildNextDynasty().SignProposer()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))

	ds = seed.Tail().State().DposState()
	inDynasty, err = ds.InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.Equal(t, true, inDynasty)

	bb = bb.Child().
		Tx().Type(dpos.TxOpQuitCandidacy).SignPair(newCandidate).Execute().
		SignProposer()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))

	ds = seed.Tail().State().DposState()
	isCandidate, err = ds.IsCandidate(newCandidate.Addr)
	require.NoError(t, err)
	assert.Equal(t, false, isCandidate)
	inDynasty, err = ds.InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.Equal(t, true, inDynasty)

	bb = bb.ChildNextDynasty().SignProposer()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))

	ds = seed.Tail().State().DposState()
	isCandidate, err = ds.IsCandidate(newCandidate.Addr)
	require.NoError(t, err)
	assert.Equal(t, false, isCandidate)
	inDynasty, err = ds.InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.Equal(t, false, inDynasty)
}
