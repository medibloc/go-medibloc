package dpos_test

import (
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/common/trie"
	dposState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	coreState "github.com/medibloc/go-medibloc/core/state"
	transaction "github.com/medibloc/go-medibloc/core/transaction"
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

	bb = bb.Child().SignProposer()
	b = bb.Build()

	dynasty, err = b.State().DposState().Dynasty()
	require.NoError(t, err)
	t.Log("new dynasty", dynasty)

}

func TestChangeDynasty(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	newCandidate := seed.Config.TokenDist[testutil.DynastySize]
	t.Log("new candidiate:", newCandidate.Addr.Hex())

	bb := blockutil.New(t, testutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist)

	bb = bb.ChildNextDynasty().
		Tx().Type(coreState.TxOpStake).Value(300000000).SignPair(newCandidate).Execute().
		Tx().Type(coreState.TxOpRegisterAlias).Value(1000000).Payload(&transaction.RegisterAliasPayload{AliasName: testutil.TestAliasName}).SignPair(newCandidate).Execute().
		Tx().Type(dposState.TxOpBecomeCandidate).Value(1000000).SignPair(newCandidate).Execute()

	acc, err := bb.Build().State().GetAccount(newCandidate.Addr)
	require.NoError(t, err)
	cId := acc.CandidateID
	require.NotNil(t, cId)

	_, err = bb.Build().State().DposState().CandidateState().Get(cId)
	require.NoError(t, err)

	votePayload := new(transaction.VotePayload)
	votePayload.CandidateIDs = append(votePayload.CandidateIDs, acc.CandidateID)
	bb = bb.
		Tx().Type(dposState.TxOpVote).
		Payload(votePayload).SignPair(newCandidate).Execute().SignProposer()
	block := bb.Build().BlockData
	err = seed.Med.BlockManager().PushBlockData(block)
	require.NoError(t, err)
	err = seed.WaitUntilBlockAcceptedOnChain(block.Hash(), 10*time.Second)
	require.NoError(t, err)
	t.Log(seed.Tail().State().DposState().Dynasty())

	ok, err := seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.False(t, ok)

	bb = bb.ChildNextDynasty().SignProposer()
	block = bb.Build().BlockData
	err = seed.Med.BlockManager().PushBlockData(block)
	require.NoError(t, err)
	err = seed.WaitUntilBlockAcceptedOnChain(block.Hash(), 10*time.Second)
	require.NoError(t, err)
	t.Log(seed.Tail().State().DposState().Dynasty())

	ok, err = seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.True(t, ok)

	bb = bb.Child().
		Tx().Type(dposState.TxOpQuitCandidacy).SignPair(newCandidate).Execute().
		SignProposer()
	block = bb.Build().BlockData
	err = seed.Med.BlockManager().PushBlockData(block)
	require.NoError(t, err)
	err = seed.WaitUntilBlockAcceptedOnChain(block.Hash(), 10*time.Second)
	require.NoError(t, err)

	acc, err = bb.Build().State().GetAccount(newCandidate.Addr)
	require.NoError(t, err)
	require.Nil(t, acc.CandidateID)

	_, err = seed.Tail().State().DposState().CandidateState().Get(cId)
	require.Error(t, trie.ErrNotFound)

	ok, err = seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.True(t, ok)

	bb = bb.ChildNextDynasty().SignProposer()
	block = bb.Build().BlockData
	err = seed.Med.BlockManager().PushBlockData(block)
	require.NoError(t, err)
	err = seed.WaitUntilBlockAcceptedOnChain(block.Hash(), 10*time.Second)
	require.NoError(t, err)

	ok, err = seed.Tail().State().DposState().InDynasty(newCandidate.Addr)
	require.NoError(t, err)
	assert.False(t, ok)
}
