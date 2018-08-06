package dpos_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeNewDynasty(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	cs := bb.Build().State().DposState().CandidateState()
	ds := bb.Build().State().DposState().DynastyState()

	genesisDynasty, err := dpos.DynastyStateToDynasty(ds)
	require.NoError(t, err)
	t.Log("genesis dynasty", genesisDynasty)

	candidates, err := dpos.SortByVotePower(cs)
	require.NoError(t, err)

	newDynasty := dpos.MakeNewDynasty(candidates, testutil.DynastySize)
	t.Log("new dynasty:", newDynasty)
}

func TestChangeDynasty(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	newCandidate := seed.Config.TokenDist[testutil.DynastySize]

	bb := blockutil.New(t, testutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist)

	bb = bb.Child().
		Tx().Type(dpos.TxOpBecomeCandidate).Value(0).SignPair(newCandidate).Execute().
		Tx().Type(core.TxOpVest).Value(10).SignPair(newCandidate).Execute().
		Tx().Type(dpos.TxOpVote).To(newCandidate.Addr).SignPair(newCandidate).Execute().SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	ds := seed.Tail().State().DposState().DynastyState()
	assert.Nil(t, testutil.KeyOf(t, ds, newCandidate.Addr.Bytes()))

	bb = bb.ChildNextDynasty().SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	ds = seed.Tail().State().DposState().DynastyState()
	assert.NotNil(t, testutil.KeyOf(t, ds, newCandidate.Addr.Bytes()))

	bb = bb.Child().
		Tx().Type(dpos.TxOpQuitCandidacy).SignPair(newCandidate).Execute().
		SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	ds = seed.Tail().State().DposState().DynastyState()
	cs := seed.Tail().State().DposState().CandidateState()
	assert.Nil(t, testutil.KeyOf(t, cs, newCandidate.Addr.Bytes()))
	assert.NotNil(t, testutil.KeyOf(t, ds, newCandidate.Addr.Bytes()))

	bb = bb.ChildNextDynasty().SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	ds = seed.Tail().State().DposState().DynastyState()
	cs = seed.Tail().State().DposState().CandidateState()
	assert.Nil(t, testutil.KeyOf(t, ds, newCandidate.Addr.Bytes()))
	assert.Nil(t, testutil.KeyOf(t, cs, newCandidate.Addr.Bytes()))

}
