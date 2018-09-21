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
	"testing"

	"strings"
	"time"

	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestBlockManager_Sequential(t *testing.T) {
	var nBlocks = 5

	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	for i := 1; i < nBlocks; i++ {
		tail := seed.Tail()
		mint := bb.Block(tail).Child().SignMiner().Build()
		require.NoError(t, bm.PushBlockData(mint.BlockData))
		assert.Equal(t, bm.TailBlock().Hash(), mint.Hash())
	}
}

func TestBlockManager_Reverse(t *testing.T) {
	var nBlocks = 5

	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	var blocks []*core.Block
	for i := 1; i < nBlocks; i++ {
		mint := bb.Block(tail).Child().SignMiner().Build()
		blocks = append(blocks, mint)
		tail = mint
	}

	for i := len(blocks) - 1; i >= 0; i-- {
		require.NoError(t, bm.PushBlockData(blocks[i].BlockData))
	}
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

	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)

	genesis := seed.Tail()
	tail := genesis
	for i := 1; i < mainChainHeight; i++ {
		var mint *core.Block
		if i == 1 {
			mint = bb.Block(tail).Child().Stake().SignMiner().Build()
		} else {
			mint = bb.Block(tail).Child().SignMiner().Build()
		}
		mainChainBlocks = append(mainChainBlocks, mint)
		tail = mint
	}

	tail = genesis
	var txs []*core.Transaction
	for i := 1; i < forkedChainHeight; i++ {
		var mint *core.Block
		if i+1 < forkedHeight {
			mint = mainChainBlocks[i-1]
		} else {
			mint = bb.Block(tail).Child().Tx().RandomTx().Execute().SignMiner().Build()
			txs = append(txs, mint.BlockData.Transactions()[0])
		}
		forkedChainBlocks = append(forkedChainBlocks, mint)
		tail = mint
	}

	tm := seed.Med.TransactionManager()
	for _, tx := range txs {
		assert.NoError(t, tm.Push(tx))
	}
	assert.Equal(t, len(txs), len(tm.GetAll()))

	bm := seed.Med.BlockManager()
	for i := len(forkedChainBlocks) - 1; i >= 0; i-- {
		assert.NoError(t, bm.PushBlockData(forkedChainBlocks[i].BlockData))
	}
	assert.Equal(t, forkedChainBlocks[len(forkedChainBlocks)-1].Hash(), seed.Tail().Hash())
	assert.Equal(t, 0, len(tm.GetAll()))

	for i := len(mainChainBlocks) - 1; i >= forkedHeight-2; i-- {
		assert.NoError(t, bm.PushBlockData(mainChainBlocks[i].BlockData))
	}
	assert.Equal(t, mainChainBlocks[len(mainChainBlocks)-1].Hash(), seed.Tail().Hash())
	assert.Equal(t, len(txs), len(tm.GetAll()))

}

func TestBlockManager_CircularParentLink(t *testing.T) {
	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	block1 := bb.Block(tail).Child().SignMiner().Build()
	block2 := bb.Block(block1).Child().SignMiner().Build()

	err := bm.PushBlockData(block1.GetBlockData())
	require.NoError(t, err)

	bb = bb.Block(block2).Child().Hash(block2.ParentHash()).ParentHash(block2.Hash())
	miner := testNetwork.FindProposer(bb.B.Timestamp(), block1)
	block3 := bb.Coinbase(miner.Addr).SignKey(miner.PrivKey).Build()

	err = bm.PushBlockData(block3.GetBlockData())
	require.Error(t, core.ErrInvalidBlockHash)
}

func TestBlockManager_FilterByLIB(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	blocks := make([]*core.Block, 0)
	for i := 0; i < dynastySize+2; i++ {
		var block *core.Block
		if i == 0 {
			block = bb.Block(tail).Child().Stake().SignMiner().Build()
		} else {
			block = bb.Block(tail).Child().SignMiner().Build()
		}
		blocks = append(blocks, block)
		require.NoError(t, bm.PushBlockData(block.GetBlockData()))
		tail = block
	}
	block := bb.Block(blocks[0]).Child().
		Tx().Type(core.TxOpAddRecord).Payload(&core.AddRecordPayload{}).SignPair(bb.KeyPairs[0]).Execute().
		SignMiner().Build()
	err := bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrCannotRevertLIB, err)

	block = bb.Block(blocks[1]).Child().
		Tx().Type(core.TxOpAddRecord).Payload(&core.AddRecordPayload{}).SignPair(bb.KeyPairs[0]).Execute().
		SignMiner().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrCannotRevertLIB, err)

	block = bb.Block(blocks[dynastySize*2/3]).Child().
		Tx().Type(core.TxOpTransfer).To(bb.KeyPairs[1].Addr).Value(10).SignPair(bb.KeyPairs[0]).Execute().
		SignMiner().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.NoError(t, err)

	block = bb.Block(blocks[dynastySize*2/3+1]).Child().
		Tx().Type(core.TxOpAddRecord).Payload(&core.AddRecordPayload{}).SignPair(bb.KeyPairs[0]).Execute().
		SignMiner().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.NoError(t, err)

	parent, err := bm.BlockByHeight(3)
	require.NoError(t, err)
	b := bb.Block(parent).Child().Height(20).
		Tx().Type(core.TxOpTransfer).To(bb.KeyPairs[1].Addr).Value(10).SignPair(bb.KeyPairs[0]).Execute().
		SignMiner().Build()
	err = bm.PushBlockData(b.GetBlockData())
	assert.Equal(t, core.ErrCannotRevertLIB, err)
}

func TestBlockManager_PruneByLIB(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	blocks := make([]*core.Block, 0)

	b1 := bb.Block(tail).Child().Stake().SignMiner().Build()
	blocks = append(blocks, b1)

	b2 := bb.Block(b1).Child().
		Tx().Type(core.TxOpAddRecord).Payload(&core.AddRecordPayload{RecordHash: []byte("recordHash0")}).SignPair(bb.KeyPairs[0]).Execute().
		SignMiner().Build()
	blocks = append(blocks, b2)

	b3 := bb.Block(b1).Child().SignMiner().Build()
	blocks = append(blocks, b3)

	for i := 1; i < dynastySize+2; i++ {
		recordPayload := &core.AddRecordPayload{
			RecordHash: []byte(fmt.Sprintf("recordHash%v", i)),
		}
		block := bb.Block(blocks[i+1]).Child().
			Tx().Type(core.TxOpAddRecord).Payload(recordPayload).SignPair(bb.KeyPairs[0]).Execute().
			SignMiner().Build()
		blocks = append(blocks, block)
	}

	for _, block := range blocks {
		err := bm.PushBlockData(block.GetBlockData())
		assert.NoError(t, err)
	}

	assert.Nil(t, bm.BlockByHash(blocks[1].Hash()))
	assert.NotNil(t, bm.BlockByHash(blocks[2].Hash()))
}

func TestBlockManager_InvalidHeight(t *testing.T) {
	dynastySize := 21
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

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
				SignMiner().Build()
		} else {
			block = bb.Block(tail).Child().
				Tx().RandomTx().Execute().
				SignMiner().Build()
		}
		err := bm.PushBlockData(block.GetBlockData())
		tail = block
		assert.NoError(t, err)
	}

	parent, err := bm.BlockByHeight(3)
	require.Nil(t, err)

	tests := []struct {
		height uint64
		err    error
	}{
		{0, core.ErrCannotRevertLIB},
		{1, core.ErrCannotRevertLIB},
		{2, core.ErrCannotExecuteOnParentBlock},
		{3, core.ErrCannotExecuteOnParentBlock},
		{5, core.ErrCannotExecuteOnParentBlock},
		{6, core.ErrCannotExecuteOnParentBlock},
		{999, core.ErrCannotExecuteOnParentBlock},
		{4, nil},
	}
	for _, v := range tests {
		block := bb.Block(parent).Child().
			Tx().RandomTx().Execute().
			Height(v.height).SignMiner().Build()
		assert.Equal(t, v.err, bm.PushBlockData(block.GetBlockData()), "testcase = %v", v)
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
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	genesis := seed.GenesisBlock()

	block := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.Dynasties).AddKeyPairs(seed.Config.TokenDist).
		Block(genesis).Child().ChainID(959123).SignMiner().Build()
	err := bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrInvalidChainID, err)
}

func TestBlockManager_RequestParentBlock(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	hook := testNetwork.SetLogTestHook()

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
		block := bb.Block(parent).Child().SignMiner().Build()
		err := bm.PushBlockData(block.GetBlockData())
		require.NoError(t, err)
		blocks = append(blocks, block)
		parent = block
	}

	seedID := seed.Med.NetService().Node().ID()

	// Invalid Protobuf
	bytes := []byte("invalid protobuf")
	node.Med.NetService().SendMsg(core.MessageTypeRequestBlock, bytes, seedID, 1)
	assert.True(t, foundInLog(hook, "Failed to unmarshal download parent block msg."))
	hook.Reset()

	// Request Genesis's Parent
	invalid := &corepb.DownloadParentBlock{
		Hash: core.GenesisHash,
		Sign: []byte{},
	}
	bytes, err := proto.Marshal(invalid)
	require.NoError(t, err)
	node.Med.NetService().SendMsg(core.MessageTypeRequestBlock, bytes, seedID, 1)
	assert.True(t, foundInLog(hook, "Asked to download genesis's parent, ignore it."))
	hook.Reset()

	// Hash Not found
	invalid = &corepb.DownloadParentBlock{
		Hash: []byte("Not Found RecordHash"),
		Sign: []byte{},
	}
	bytes, err = proto.Marshal(invalid)
	require.NoError(t, err)
	node.Med.NetService().SendMsg(core.MessageTypeRequestBlock, bytes, seedID, 1)
	assert.True(t, foundInLog(hook, "Failed to find the block asked for."))
	hook.Reset()

	// Sign mismatch
	invalid = &corepb.DownloadParentBlock{
		Hash: blocks[4].Hash(),
		Sign: []byte("Invalid signature"),
	}
	bytes, err = proto.Marshal(invalid)
	require.NoError(t, err)
	node.Med.NetService().SendMsg(core.MessageTypeRequestBlock, bytes, seedID, 1)
	assert.True(t, foundInLog(hook, "Failed to check the block's signature."))
	hook.Reset()
}

func TestBlockManager_VerifyIntegrity(t *testing.T) {
	dynastySize := testutil.DynastySize
	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()
	testNetwork.SetLogTestHook()
	seed := testNetwork.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	//dynasties := nt.Seed.Config.Dynasties
	genesis := seed.GenesisBlock()
	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.TokenDist).AddKeyPairs(seed.Config.Dynasties)

	// Invalid Block Hash
	bb = bb.Block(genesis).Child()
	pair := testNetwork.FindProposer(bb.B.Timestamp(), genesis)
	block := bb.Coinbase(pair.Addr).PayReward().Flush().Seal().Hash(hash([]byte("invalid hash"))).SignKey(pair.PrivKey).Build()
	err := bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrInvalidBlockHash, err)

	// Invalid Block Sign algorithm
	block = bb.Block(genesis).Child().SignMiner().Alg(11).Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrCannotExecuteOnParentBlock, err)

	// Invalid Block Signer
	invalidPair := testutil.NewAddrKeyPair(t)
	block = bb.Block(genesis).Child().SignPair(invalidPair).Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrCannotExecuteOnParentBlock, err)

	// Invalid Transaction Hash
	pair = testutil.NewAddrKeyPair(t)
	block = bb.Block(genesis).Child().Tx().Hash(hash([]byte("invalid hash"))).From(pair.Addr).SignKey(pair.PrivKey).Add().SignMiner().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrInvalidTransactionHash, err)

	// Invalid Transaction Signer
	pair1 := testutil.NewAddrKeyPair(t)
	pair2 := testutil.NewAddrKeyPair(t)
	block = bb.Block(genesis).Child().Tx().From(pair1.Addr).CalcHash().SignKey(pair2.PrivKey).Add().SignMiner().Build()
	err = bm.PushBlockData(block.GetBlockData())
	assert.Equal(t, core.ErrInvalidTransactionSigner, err)
}

func TestBlockManager_InvalidState(t *testing.T) {
	dynastySize := testutil.DynastySize
	tn := testutil.NewNetwork(t, dynastySize)
	defer tn.Cleanup()
	tn.SetLogTestHook()
	seed := tn.NewSeedNode()
	seed.Start()
	bm := seed.Med.BlockManager()

	genesis := seed.GenesisBlock()
	bb := blockutil.New(t, dynastySize).AddKeyPairs(seed.Config.TokenDist).AddKeyPairs(seed.Config.Dynasties)

	from := seed.Config.TokenDist[0]
	to := seed.Config.TokenDist[1]
	bb = bb.Block(genesis).Child().Stake().
		Tx().Type(core.TxOpTransfer).To(to.Addr).Value(100).SignPair(from).Execute().
		Tx().Type(core.TxOpAddRecord).
		Payload(&core.AddRecordPayload{
			RecordHash: hash([]byte("Record Hash")),
		}).SignPair(from).Execute().
		Tx().Type(core.TxOpAddCertification).To(to.Addr).
		Payload(&core.AddCertificationPayload{
			IssueTime:       time.Now().Unix(),
			ExpirationTime:  time.Now().Add(24 * time.Hour * 365).Unix(),
			CertificateHash: hash([]byte("Certificate Root Hash")),
		}).SignPair(from).Execute().
		Tx().Type(core.TxOpRevokeCertification).To(to.Addr).
		Payload(&core.RevokeCertificationPayload{
			CertificateHash: hash([]byte("Certificate Root Hash")),
		}).SignPair(from).Execute().
		Tx().Type(core.TxOpVest).Value(100).SignPair(from).Execute().
		Tx().Type(core.TxOpWithdrawVesting).Value(100).SignPair(from).Execute()

	miner := bb.FindMiner()
	bb = bb.Coinbase(miner.Addr).PayReward().Flush()

	block := bb.Clone().AccountRoot(hash([]byte("invalid account root"))).CalcHash().SignKey(miner.PrivKey).Build()
	err := bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrCannotExecuteOnParentBlock, err)

	block = bb.Clone().TxRoot(hash([]byte("invalid txs root"))).CalcHash().SignKey(miner.PrivKey).Build()
	err = bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrCannotExecuteOnParentBlock, err)

	block = bb.Clone().DposRoot(hash([]byte("invalid dpos root"))).CalcHash().SignKey(miner.PrivKey).Build()
	err = bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrCannotExecuteOnParentBlock, err)

	block = bb.Clone().Timestamp(time.Now().Add(11111 * time.Second).Unix()).CalcHash().SignKey(miner.PrivKey).Build()
	err = bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrCannotExecuteOnParentBlock, err)

	block = bb.Clone().ChainID(1111111).CalcHash().SignKey(miner.PrivKey).Build()
	err = bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrInvalidChainID, err)

	block = bb.Clone().Tx().Type(core.TxOpVest).Value(100).SignPair(from).Execute().CalcHash().SignKey(miner.PrivKey).Alg(111).Build()
	err = bm.PushBlockData(block.GetBlockData())
	require.Equal(t, core.ErrCannotExecuteOnParentBlock, err)

	block = bb.Clone().Coinbase(to.Addr).Seal().CalcHash().SignKey(miner.PrivKey).Build()
	err = bm.PushBlockData(block.GetBlockData())
	require.NoError(t, err)
}

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
