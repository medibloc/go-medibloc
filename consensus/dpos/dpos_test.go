package dpos_test

import (
	"context"
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/crypto/hash"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeDynasty(t *testing.T) {
	const (
		pushTimeLimit = 10 * time.Second
	)

	testNetwork := testutil.NewNetwork(t)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bm := seed.Med.BlockManager()

	newCandidate := seed.Config.TokenDist[blockutil.DynastySize]
	t.Log("new candidiate:", newCandidate.Addr.Hex())

	bb := blockutil.New(t, blockutil.DynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist)

	// Become new candidate
	bb = bb.ChildNextDynasty().
		Tx().Type(transaction.TxOpStake).Value(100000001).SignPair(newCandidate).Execute().
		Tx().Type(transaction.TxOpRegisterAlias).Value(1000000).Payload(&transaction.RegisterAliasPayload{AliasName: "newblockproducer"}).SignPair(newCandidate).Execute().
		Tx().Type(transaction.TxOpBecomeCandidate).Value(1000000).SignPair(newCandidate).Execute()

	acc, err := bb.Build().State().GetAccount(newCandidate.Addr)
	require.NoError(t, err)
	cId := acc.CandidateID
	require.NotNil(t, cId)

	_, err = bb.Build().State().DposState().GetCandidate(cId)
	require.NoError(t, err)

	// Self vote
	votePayload := new(transaction.VotePayload)
	votePayload.CandidateIDs = append(votePayload.CandidateIDs, acc.CandidateID)
	bb = bb.
		Tx().Type(transaction.TxOpVote).
		Payload(votePayload).SignPair(newCandidate).Execute().SignProposer()
	bd := bb.Build().BlockData

	ctx, cancel := context.WithTimeout(context.Background(), pushTimeLimit)
	defer cancel()
	require.NoError(t, bm.PushBlockDataSync2(ctx, bd))
	require.NoError(t, seed.WaitUntilBlockAcceptedOnChain(bd.Hash(), 10*time.Second))

	t.Log(dynasty(t, bm))
	assert.False(t, inDynasty(t, bm, newCandidate.Addr)) // new candidate are going to be producer in next dynasty

	// wait for next dynasty
	bb = bb.ChildNextDynasty().SignProposer()
	bd = bb.Build().BlockData
	ctx, cancel = context.WithTimeout(context.Background(), pushTimeLimit)
	defer cancel()
	require.NoError(t, bm.PushBlockDataSync2(ctx, bd))
	require.NoError(t, seed.WaitUntilBlockAcceptedOnChain(bd.Hash(), 10*time.Second))

	t.Log(dynasty(t, bm))
	assert.True(t, inDynasty(t, bm, newCandidate.Addr)) // new candidate become producer

	// quit candidate
	bb = bb.Child().
		Tx().Type(transaction.TxOpQuitCandidacy).SignPair(newCandidate).Execute().
		SignProposer()
	bd = bb.Build().BlockData
	ctx, cancel = context.WithTimeout(context.Background(), pushTimeLimit)
	defer cancel()
	require.NoError(t, bm.PushBlockDataSync2(ctx, bd))
	require.NoError(t, seed.WaitUntilBlockAcceptedOnChain(bd.Hash(), 10*time.Second))

	t.Log(dynasty(t, bm))
	acc, err = bb.Build().State().GetAccount(newCandidate.Addr)
	require.NoError(t, err)
	require.Nil(t, acc.CandidateID)

	_, err = seed.Tail().State().DposState().GetCandidate(cId)
	require.Error(t, trie.ErrNotFound)

	assert.True(t, inDynasty(t, bm, newCandidate.Addr)) // still in producer

	bb = bb.ChildNextDynasty().SignProposer()
	bd = bb.Build().BlockData
	ctx, cancel = context.WithTimeout(context.Background(), pushTimeLimit)
	defer cancel()
	require.NoError(t, bm.PushBlockDataSync2(ctx, bd))

	require.NoError(t, seed.WaitUntilBlockAcceptedOnChain(bd.Hash(), 10*time.Second))
	assert.False(t, inDynasty(t, bm, newCandidate.Addr))
}

func TestMakeMintBlock(t *testing.T) {
	dynastySize := 21

	testNetwork := testutil.NewNetworkWithDynastySize(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.AddProposers(seed.Config.Dynasties)
	seed.Start()

	testNetwork.WaitForEstablished()

	td := seed.Config.TokenDist
	payer := td[0]

	bb := blockutil.New(t, dynastySize).Block(seed.Tail()).AddKeyPairs(seed.Config.TokenDist)
	tb := bb.Tx()

	randomSender := keyutil.NewAddrKeyPair(t)
	payload := &transaction.AddRecordPayload{RecordHash: hash.Sha3256([]byte("testRecord"))}
	tx := tb.Type(transaction.TxOpAddRecord).Payload(payload).SignPair(randomSender).SignPayerPair(payer).Build()

	tm := seed.Med.TransactionManager()
	require.NoError(t, tm.Push(tx))

	for tm.Get(tx.Hash()) != nil {
		time.Sleep(3 * time.Second)
	}

	txInChain, err := seed.Tail().State().GetTx(tx.Hash())
	require.NoError(t, err)

	height := txInChain.Receipt().Height()

	parent, err := seed.Med.BlockManager().BlockByHeight(height - 1)
	require.NoError(t, err)

	b, err := seed.Med.BlockManager().BlockByHeight(height)
	require.NoError(t, err)

	_, err = parent.CreateChildWithBlockData(b.BlockData, seed.Med.BlockManager().Consensus())
	require.NoError(t, err)
}

func dynasty(t *testing.T, bm *core.BlockManager) []common.Address {
	block := bm.TailBlock()
	dynastySize := bm.Consensus().DynastySize()
	dynasty := make([]common.Address, dynastySize)

	var err error
	for i := 0; i < dynastySize; i++ {
		dynasty[i], err = block.State().DposState().GetProposer(i)
		require.NoError(t, err)
	}
	return dynasty
}

func inDynasty(t *testing.T, bm *core.BlockManager, address common.Address) bool {
	block := bm.TailBlock()
	dynastySize := bm.Consensus().DynastySize()

	for i := 0; i < dynastySize; i++ {
		proposer, err := block.State().DposState().GetProposer(i)
		require.NoError(t, err)
		if proposer.Equals(address) {
			return true
		}
	}
	return false
}
