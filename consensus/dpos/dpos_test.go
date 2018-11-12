package dpos_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/common/trie"
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

	dynasty, err := b.State().DposState().Dynasty()
	require.NoError(t, err)
	t.Log("genesis dynasty", dynasty)

	bb = bb.Child().SignMiner()
	b = bb.Build()

	dynasty, err = b.State().DposState().Dynasty()
	require.NoError(t, err)
	t.Log("new dynasty", dynasty)

}

func TestChangeDynasty(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	newCandidate := seed.Config.TokenDist[testutil.DynastySize]
	t.Log("new candidiate:", newCandidate.Addr.Hex())

	bb := blockutil.New(t, testutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist)

	bb = bb.ChildNextDynasty().
		Tx().Type(core.TxOpVest).Value(300000000).SignPair(newCandidate).Execute().
		Tx().Type(dpos.TxOpBecomeCandidate).Value(1000000).SignPair(newCandidate).Execute()

	acc, err := bb.Build().State().GetAccount(newCandidate.Addr)
	require.NoError(t, err)
	cId := acc.CandidateID
	require.NotNil(t, cId)

	_, err = bb.Build().State().DposState().CandidateState().Get(cId)
	require.NoError(t, err)

	votePayload := new(dpos.VotePayload)
	votePayload.CandidateIDs = append(votePayload.CandidateIDs, acc.CandidateID)
	bb = bb.
		Tx().Type(dpos.TxOpVote).
		Payload(votePayload).SignPair(newCandidate).Execute().SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	t.Log(seed.Tail().State().DposState().Dynasty())

	ok, err := seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.False(t, ok)

	bb = bb.ChildNextDynasty().SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	t.Log(seed.Tail().State().DposState().Dynasty())

	ok, err = seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.True(t, ok)

	bb = bb.Child().
		Tx().Type(dpos.TxOpQuitCandidacy).SignPair(newCandidate).Execute().
		SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))

	acc, err = bb.Build().State().GetAccount(newCandidate.Addr)
	require.NoError(t, err)
	require.Nil(t, acc.CandidateID)

	_, err = seed.Tail().State().DposState().CandidateState().Get(cId)
	require.Error(t, trie.ErrNotFound)

	ok, err = seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.True(t, ok)

	bb = bb.ChildNextDynasty().SignMiner()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))

	ok, err = seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.False(t, ok)
}
