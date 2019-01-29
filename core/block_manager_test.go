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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestBlockManager_Sequential(t *testing.T) {
	var nBlocks = 5

	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	tail := seed.Tail()
	for i := 1; i < nBlocks; i++ {
		tail = bb.Block(tail).Child().SignProposer().Build()
		err := bm.PushBlockData(tail.BlockData)
		assert.NoError(t, err)
		err = seed.WaitUntilTailHeight(tail.Height(), 10*time.Second)
		assert.NoError(t, err)
		assert.Equal(t, bm.TailBlock().Hash(), tail.Hash())
	}
}

func TestBlockManager_Reverse(t *testing.T) {
	var nBlocks = 5

	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	var blocks []*core.Block
	for i := 1; i < nBlocks; i++ {
		mint := bb.Block(tail).Child().SignProposer().Build()
		blocks = append(blocks, mint)
		tail = mint
	}

	for i := len(blocks) - 1; i >= 0; i-- {
		require.NoError(t, bm.PushBlockData(blocks[i].BlockData))
	}
	require.NoError(t, seed.WaitUntilTailHeight(uint64(nBlocks), 10*time.Second))
	assert.Equal(t, nBlocks, int(bm.TailBlock().Height()))
}

func TestBlockManager_Forked(t *testing.T) {
	var (
		mainChainHeight   = 6
		forkedChainHeight = 5
		forkedHeight      = 3 // must higher than LIB height of forkedHeight
		mainChainBlocks   []*core.Block
		forkedChainBlocks []*core.Block
	)

	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	genesis := seed.Tail()
	tail := genesis
	for i := 1; i < mainChainHeight; i++ {
		var mint *core.Block
		if i == 1 {
			mint = bb.Block(tail).Child().Stake().SignProposer().Build()
		} else {
			mint = bb.Block(tail).Child().SignProposer().Build()
		}
		mainChainBlocks = append(mainChainBlocks, mint)
		tail = mint
	}

	tail = genesis
	var txs []*Transaction
	for i := 1; i < forkedChainHeight; i++ {
		var mint *core.Block
		if i+1 < forkedHeight {
			mint = mainChainBlocks[i-1]
		} else {
			mint = bb.Block(tail).Child().Tx().RandomTx().Execute().SignProposer().Build()
			txs = append(txs, mint.BlockData.Transactions()[0])
		}
		forkedChainBlocks = append(forkedChainBlocks, mint)
		tail = mint
	}

	tm := seed.Med.TransactionManager()
	failed := tm.PushAndExclusiveBroadcast(txs...)
	assert.Zero(t, len(failed))
	assert.Equal(t, len(txs), len(tm.GetAll()))

	bm := seed.Med.BlockManager()
	for i := len(forkedChainBlocks) - 1; i >= 0; i-- {
		assert.NoError(t, bm.PushBlockData(forkedChainBlocks[i].BlockData))
	}
	assert.NoError(t, seed.WaitUntilTailHeight(uint64(forkedChainHeight), 10*time.Second))
	assert.Equal(t, forkedChainBlocks[len(forkedChainBlocks)-1].Hash(), seed.Tail().Hash())
	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, len(tm.GetAll()))

	for i := len(mainChainBlocks) - 1; i >= forkedHeight-2; i-- {
		err := bm.PushBlockData(mainChainBlocks[len(mainChainBlocks)-i].BlockData)
		assert.NoError(t, err)
	}
	assert.NoError(t, seed.WaitUntilTailHeight(uint64(mainChainHeight), 10*time.Second))
	assert.Equal(t, mainChainBlocks[len(mainChainBlocks)-1].Hash(), seed.Tail().Hash())
	time.Sleep(1 * time.Second)
	assert.Equal(t, len(txs), len(tm.GetAll()))
}

func TestBlockManager_CircularParentLink(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	block1 := bb.Block(tail).Child().SignProposer().Build()
	block2 := bb.Block(block1).Child().SignProposer().Build()

	err := bm.PushBlockData(block1.GetBlockData())
	require.NoError(t, err)
	err = seed.WaitUntilTailHeight(block1.Height(), 10*time.Second)
	require.NoError(t, err)

	bb = bb.Block(block2).Child().Hash(block2.ParentHash()).ParentHash(block2.Hash())
	proposer := testNetwork.FindProposer(bb.B.Timestamp(), block1)
	block3 := bb.Coinbase(proposer.Addr).SignKey(proposer.PrivKey).Build()

	err = bm.PushBlockData(block3.GetBlockData())
	require.Equal(t, core.ErrDuplicatedBlock, err)
}

func TestBlockManager_FilterByLIB(t *testing.T) {
	dynastySize := blockutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	blocks := make([]*core.Block, 0)
	for i := 0; i < dynastySize+2; i++ {
		var block *core.Block
		if i == 0 {
			block = bb.Block(tail).Child().Stake().SignProposer().Build()
		} else {
			block = bb.Block(tail).Child().SignProposer().Build()
		}
		blocks = append(blocks, block)
		err := bm.PushBlockData(block.GetBlockData())
		require.NoError(t, err)
		tail = block
	}
	err := seed.WaitUntilTailHeight(tail.Height(), 10*time.Second)
	require.NoError(t, err)

	recordHash, err := byteutils.Hex2Bytes("255607ec7ef55d7cfd8dcb531c4aa33c4605f8aac0f5784a590041690695e6f7")
	require.NoError(t, err)
	payload := &transaction.AddRecordPayload{
		RecordHash: recordHash,
	}
	block := bb.Block(blocks[0]).Child().
		Tx().Type(coreState.TxOpAddRecord).Payload(payload).SignPair(bb.KeyPairs[0]).Execute().
		SignProposer().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrFailedValidateHeightAndHeight, err)

	block = bb.Block(blocks[1]).Child().
		Tx().Type(coreState.TxOpAddRecord).Payload(payload).SignPair(bb.KeyPairs[0]).Execute().
		SignProposer().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrFailedValidateHeightAndHeight, err)

	block = bb.Block(blocks[dynastySize*2/3]).Child().
		Tx().Type(coreState.TxOpTransfer).To(bb.KeyPairs[1].Addr).Value(10).SignPair(bb.KeyPairs[0]).Execute().
		SignProposer().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.NoError(t, err)

	block = bb.Block(blocks[dynastySize*2/3+1]).Child().
		Tx().Type(coreState.TxOpAddRecord).Payload(payload).SignPair(bb.KeyPairs[0]).Execute().
		SignProposer().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.NoError(t, err)

	parent, err := bm.BlockByHeight(3)
	require.NoError(t, err)
	b := bb.Block(parent).Child().
		Tx().Type(coreState.TxOpTransfer).To(bb.KeyPairs[1].Addr).Value(10).SignPair(bb.KeyPairs[0]).Execute().
		SignProposer().Build()
	err = bm.PushBlockData(b.GetBlockData())
	assert.Equal(t, core.ErrFailedValidateHeightAndHeight, err)
}

func TestBlockManager_PruneByLIB(t *testing.T) {
	dynastySize := blockutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	blocks := make([]*core.Block, 0)

	b1 := bb.Block(tail).Child().Stake().SignProposer().Build()
	blocks = append(blocks, b1)

	recordHash, _ := byteutils.Hex2Bytes("255607ec7ef55d7cfd8dcb531c4aa33c4605f8aac0f5784a590041690695e6f7")
	payload := &transaction.AddRecordPayload{
		RecordHash: recordHash,
	}
	b2 := bb.Block(b1).Child().
		Tx().Type(coreState.TxOpAddRecord).Payload(payload).SignPair(bb.KeyPairs[0]).Execute().
		SignProposer().Build()
	blocks = append(blocks, b2)

	b3 := bb.Block(b1).Child().SignProposer().Build()
	blocks = append(blocks, b3)

	for i := 1; i < dynastySize+2; i++ {
		hash := fmt.Sprintf("255607ec7ef55d7cfd8dcb531c4aa33c4605f8aac0f5784a590041690695e6f%v", i)
		recordHash, _ := byteutils.Hex2Bytes(hash)
		payload := &transaction.AddRecordPayload{
			RecordHash: recordHash,
		}
		block := bb.Block(blocks[i+1]).Child().
			Tx().Type(coreState.TxOpAddRecord).Payload(payload).SignPair(bb.KeyPairs[0]).Execute().
			SignProposer().Build()
		blocks = append(blocks, block)
	}

	for _, block := range blocks {
		err := bm.PushBlockData(block.GetBlockData())
		assert.NoError(t, err)
	}

	assert.NoError(t, seed.WaitUntilTailHeight(blocks[len(blocks)-1].Height(), 10*time.Second))
	block, err := bm.BlockByHeight(3)
	require.NoError(t, err)
	assert.NotEqual(t, blocks[1].Hash(), block.Hash())
	assert.Equal(t, blocks[2].Hash(), block.Hash())
}

func TestBlockManager_InvalidHeight(t *testing.T) {
	dynastySize := 21
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	tail := seed.Tail()
	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	for i := 0; i < 6; i++ {
		var block *core.Block
		if i == 0 {
			block = bb.Block(tail).Child().Stake().
				Tx().RandomTx().Execute().
				SignProposer().Build()
		} else {
			block = bb.Block(tail).Child().
				Tx().RandomTx().Execute().
				SignProposer().Build()
		}
		err := bm.PushBlockData(block.GetBlockData())
		assert.NoError(t, err)
		tail = block
	}
	err := seed.WaitUntilTailHeight(tail.Height(), 10*time.Second)
	assert.NoError(t, err)
	parent, err := bm.BlockByHeight(3)
	require.Nil(t, err)

	tests := []struct {
		height uint64
		err    error
	}{
		{0, core.ErrFailedValidateHeightAndHeight},
		{1, core.ErrFailedValidateHeightAndHeight},
		{2, core.ErrInvalidBlock},
		{3, core.ErrInvalidBlock},
		{5, core.ErrFailedValidateHeightAndHeight},
		{6, core.ErrFailedValidateHeightAndHeight},
		{999, core.ErrFailedValidateHeightAndHeight},
		{4, nil},
	}
	for _, v := range tests {
		block := bb.Block(parent).Child().
			Tx().RandomTx().Execute().
			Height(v.height).SignProposer().Build()
		assert.Equal(t, v.err, bm.PushBlockDataSync(block.GetBlockData(), 10*time.Second), "testcase = %v", v)
	}
}

func TestBlockManager_Setup(t *testing.T) {
	cfg := medlet.DefaultConfig()
	cfg.Chain.BlockPoolSize = 0
	bm, err := core.NewBlockManager(cfg)
	require.Nil(t, bm)
	require.EqualError(t, err, "Must provide a positive size")

	cfg = medlet.DefaultConfig()
	cfg.Chain.BlockCacheSize = 0
	bm, err = core.NewBlockManager(cfg)
	require.Nil(t, bm)
	require.EqualError(t, err, "Must provide a positive size")

	cfg = medlet.DefaultConfig()
	cfg.Chain.TailCacheSize = 0
	bm, err = core.NewBlockManager(cfg)
	require.Nil(t, bm)
	require.EqualError(t, err, "Must provide a positive size")
}

func TestBlockManager_InvalidChainID(t *testing.T) {
	dynastySize := blockutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	genesis := seed.GenesisBlock()

	block := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist).
		Block(genesis).Child().ChainID(959123).SignProposer().Build()
	err := bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrInvalidBlockChainID, err)
}

func TestBlockManager_RequestParentBlock(t *testing.T) {
	dynastySize := blockutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	hook := testNetwork.LogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()
	node := testNetwork.NewNode()

	testNetwork.Start()

	testNetwork.WaitForEstablished()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	genesis := seed.GenesisBlock()

	blocks := make([]*core.Block, 0)
	parent := genesis
	for i := 0; i < 10; i++ {
		block := bb.Block(parent).Child().SignProposer().Build()
		err := bm.PushBlockData(block.GetBlockData())
		require.NoError(t, err)
		blocks = append(blocks, block)
		parent = block
	}

	seedID := seed.Med.NetService().Node().ID()

	// Invalid Protobuf
	bytes := []byte("invalid protobuf")
	node.Med.NetService().SendMessageToPeer(core.MessageTypeRequestBlock, bytes, 1, seedID.Pretty())
	assert.True(t, foundInLog(hook, "Failed to unmarshal download parent block msg."))
	hook.Reset()

	// Request Genesis's Parent
	invalid := &corepb.DownloadParentBlock{
		Hash: genesis.Hash(),
		Sign: genesis.Sign(),
	}
	bytes, err := proto.Marshal(invalid)
	require.NoError(t, err)
	node.Med.NetService().SendMessageToPeer(core.MessageTypeRequestBlock, bytes, 1, seedID.Pretty())
	assert.True(t, foundInLog(hook, "Asked to download genesis's parent, ignore it."))
	hook.Reset()

	// Hash Not found
	invalid = &corepb.DownloadParentBlock{
		Hash: []byte("Not Found RecordHash"),
		Sign: []byte{},
	}
	bytes, err = proto.Marshal(invalid)
	require.NoError(t, err)
	node.Med.NetService().SendMessageToPeer(core.MessageTypeRequestBlock, bytes, 1, seedID.Pretty())
	assert.True(t, foundInLog(hook, "Failed to find the block asked for."))
	hook.Reset()

	// Sign mismatch
	invalid = &corepb.DownloadParentBlock{
		Hash: blocks[4].Hash(),
		Sign: []byte("Invalid signature"),
	}
	bytes, err = proto.Marshal(invalid)
	require.NoError(t, err)
	node.Med.NetService().SendMessageToPeer(core.MessageTypeRequestBlock, bytes, 1, seedID.Pretty())
	assert.True(t, foundInLog(hook, "Failed to check the block's signature."))
	hook.Reset()
}

func TestBlockManager_VerifyIntegrity(t *testing.T) {
	dynastySize := blockutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	// dynasties := nt.Seed.Config.Dynasties
	genesis := seed.GenesisBlock()
	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.TokenDist).AddKeyPairs(seed.Config.Dynasties)

	// Invalid Block Hash
	bb = bb.Block(genesis).Child()
	pair := testNetwork.FindProposer(bb.B.Timestamp(), genesis)
	block := bb.Coinbase(pair.Addr).PayReward().Flush().Seal().Hash(hash([]byte("invalid hash"))).SignKey(pair.PrivKey).Build()
	err := bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Equal(t, core.ErrInvalidBlockHash, err)

	// Invalid Block Signer
	invalidPair := keyutil.NewAddrKeyPair(t)
	block = bb.Block(genesis).Child().SignPair(invalidPair).Build()
	err = bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Equal(t, core.ErrInvalidBlock, err)

	// Invalid Transaction Hash
	pair = keyutil.NewAddrKeyPair(t)
	block = bb.Block(genesis).Child().Tx().Hash(hash([]byte("invalid hash"))).SignKey(pair.PrivKey).Add().SignProposer().Build()
	err = bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Equal(t, ErrInvalidTransactionHash, err)

	// // Invalid Transaction Signer
	// pair1 := testutil.NewAddrKeyPair(t)
	// pair2 := testutil.NewAddrKeyPair(t)
	// block = bb.Block(genesis).InitChild().Tx().From(pair1.Addr).CalcHash().SignKey(pair2.PrivKey).Add().SignProposer().Build()
	// err = bm.PushBlockData(block.GetBlockData())
	// assert.Equal(t, core.ErrInvalidTransactionSigner, err)
}

func TestBlockManager_InvalidState(t *testing.T) {
	dynastySize := blockutil.DynastySize
	tn := testutil.NewNetwork(t, dynastySize)
	defer tn.Cleanup()
	seed := tn.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	genesis := seed.GenesisBlock()
	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.TokenDist).AddKeyPairs(seed.Config.Dynasties)

	from := seed.Config.TokenDist[dynastySize]
	to := seed.Config.TokenDist[dynastySize+1]
	bb = bb.Block(genesis).Child().Stake().
		Tx().Type(coreState.TxOpTransfer).To(to.Addr).Value(100).SignPair(from).Execute().
		Tx().Type(coreState.TxOpAddRecord).
		Payload(&transaction.AddRecordPayload{
			RecordHash: hash([]byte("Record Hash")),
		}).SignPair(from).Execute().
		Tx().Type(coreState.TxOpAddCertification).To(to.Addr).
		Payload(&transaction.AddCertificationPayload{
			IssueTime:       time.Now().Unix(),
			ExpirationTime:  time.Now().Add(24 * time.Hour * 365).Unix(),
			CertificateHash: hash([]byte("Certificate Root Hash")),
		}).SignPair(from).Execute().
		Tx().Type(coreState.TxOpRevokeCertification).To(to.Addr).
		Payload(&transaction.RevokeCertificationPayload{
			CertificateHash: hash([]byte("Certificate Root Hash")),
		}).SignPair(from).Execute().
		Tx().Type(coreState.TxOpStake).Value(100).SignPair(from).Execute().
		Tx().Type(coreState.TxOpUnstake).Value(100).SignPair(from).Execute()

	proposer := bb.FindProposer()
	bb = bb.Coinbase(proposer.Addr).PayReward().Flush().Seal()

	block := bb.Clone().AccountRoot(hash([]byte("invalid account root"))).CalcHash().SignKey(proposer.PrivKey).Build()
	err := bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Error(t, err, testutil.ErrExecutionTimeout)

	block = bb.Clone().TxRoot(hash([]byte("invalid txs root"))).CalcHash().SignKey(proposer.PrivKey).Build()
	err = bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Error(t, core.ErrInvalidBlock, err)

	block = bb.Clone().DposRoot(hash([]byte("invalid dpos root"))).CalcHash().SignKey(proposer.PrivKey).Build()
	err = bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Error(t, core.ErrInvalidBlock, err)

	block = bb.Clone().Timestamp(time.Now().Add(11111 * time.Second).Unix()).CalcHash().SignKey(proposer.PrivKey).Build()
	err = bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Error(t, core.ErrInvalidBlock, err)

	block = bb.Clone().ChainID(1111111).CalcHash().SignKey(proposer.PrivKey).Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrInvalidBlockChainID, err)

	block = bb.Clone().Coinbase(to.Addr).CalcHash().SignKey(proposer.PrivKey).Build()
	err = bm.PushBlockDataSync(block.GetBlockData(), 1*time.Second)
	assert.Error(t, core.ErrInvalidBlock, err)
}

func TestBlockManager_PushBlockDataSync(t *testing.T) {
	var nBlocks = 5

	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	tail := seed.Tail()
	for i := 1; i < nBlocks; i++ {
		mint := bb.Block(tail).Child().SignProposer().Build()
		err := bm.PushBlockDataSync(mint.BlockData, 10*time.Second)
		assert.NoError(t, err)
		assert.Equal(t, bm.BlockByHash(mint.Hash()).Height(), mint.Height())
		tail = mint
	}
	err := seed.WaitUntilTailHeight(tail.Height(), 10*time.Second)
	assert.NoError(t, err)

	// Timeout test
	tail = seed.Tail()
	parentBlock := bb.Block(tail).Child().SignProposer().Build()
	gappedBlock := bb.Block(parentBlock).Child().SignProposer().Build()
	err = bm.PushBlockDataSync(gappedBlock.BlockData, 1*time.Second)
	assert.Equal(t, core.ErrBlockExecutionTimeout, err)

	// Holding channel in the block pool test
	childBlock := bb.Block(gappedBlock).Child().SignProposer().Build()
	go func() {
		err := bm.PushBlockDataSync(childBlock.BlockData, 1*time.Second)
		assert.NoError(t, err)
	}()
	go func() {
		err := bm.PushBlockDataSync(parentBlock.BlockData, 1*time.Second)
		assert.NoError(t, err)
	}()

	// Execution error sent via execCh test
	invalidBlock := bb.Block(childBlock).Child().Height(tail.Height() + 2).SignProposer().Build()
	err = bm.PushBlockDataSync(invalidBlock.BlockData, 10*time.Second)
	assert.Equal(t, core.ErrInvalidBlock, err)
}

/*
func TestBlockManagerImprovement(t *testing.T) {
	dynastySize := testutil.DynastySize
	tn := testutil.NewNetwork(t, dynastySize)
	defer tn.Cleanup()
	tn.SetLogTestHook()
	seed1 := tn.NewSeedNode()
	seed1.Start()

	bm := seed1.Med.BlockManager()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed1.Config.TokenDist).AddKeyPairs(seed1.Config.Dynasties).Block(
		seed1.GenesisBlock()).ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix())).Stake().SignProposer()

	block := bb.Build()
	assert.NoError(t, bm.PushBlockData(block.BlockData))
	assert.NoError(t, seed1.WaitUntilBlockAcceptedOnChain(block.Hash()))

	blocks := make([]*core.Block, 100)
	for i := 0; i < 100; i++ {
		bb = bb.InitChild()
		for j := 0; j < 100; j++ {
			bb = bb.Tx().RandomTx().Execute()
		}
		blocks[i] = bb.SignProposer().Build()
	}

	//Sequential Test
	start := time.Now()
	for _, bbb := range blocks {
		assert.NoError(t, bm.PushBlockData(bbb.BlockData))
		seed1.WaitUntilBlockAcceptedOnChain(bbb.Hash()
	}
	sequentialElapsed := time.Since(start).Seconds()
	t.Log("SEQUENTIAL : ", sequentialElapsed)

	seed2 := tn.NewSeedNode()
	seed2.Start()

	bm = seed2.Med.BlockManager()
	bb = blockutil.New(t, dynastySize).AddKeyPairs(seed2.Config.TokenDist).AddKeyPairs(seed2.Config.Dynasties).Block(
		seed2.GenesisBlock()).ChildWithTimestamp(dpos.NextMintSlot2(time.Now().Unix())).Stake().SignProposer()

	block = bb.Build()
	assert.NoError(t, bm.PushBlockData(block.BlockData))
	assert.NoError(t, seed2.WaitUntilBlockAcceptedOnChain(block.Hash()))

	blocks = make([]*core.Block, 100)
	for i := 0; i < 100; i++ {
		bbT := bb.InitChild()
		for j := 0; j < 100; j++ {
			bbT = bbT.Tx().RandomTx().Execute()
		}
		blocks[i] = bbT.SignProposer().Build()
		if i == 99 {
			blocks = append(blocks, bbT.InitChild().SignProposer().Build())
		}
	}

	start = time.Now()
	for _, bbb := range blocks {
		assert.NoError(t, bm.PushBlockData(bbb.BlockData))
	}
	seed2.WaitUntilTailHeight(4)
	parallelElapsed := time.Since(start).Seconds()
	t.Log("PARALLEL : ", parallelElapsed)
}
*/

func foundInLog(hook *test.Hook, s string) bool {
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		for _, entry := range hook.AllEntries() {
			if strings.Contains(entry.Message, s) {
				return true
			}
		}

		time.Sleep(10 * time.Millisecond)
		select {
		case <-timer.C:
			return false
		default:
		}
	}
}

func hash(data []byte) []byte {
	hasher := sha3.New256()
	hasher.Write(data)
	return hasher.Sum(nil)
}
