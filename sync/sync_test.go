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

package sync_test

import (
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Start(t *testing.T) {
	const (
		nBlocks = 100
	)
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist).Block(seed.Tail())
	for i := 1; i < nBlocks; i++ {
		bb = bb.Child().Tx().RandomTx().Execute().SignProposer()
		err := seed.Med.BlockManager().PushBlockData(bb.Build().BlockData)
		require.NoError(t, err)
	}

	require.NoError(t, seed.WaitUntilTailHeight(nBlocks, 10*time.Second))
	require.Equal(t, uint64(nBlocks), seed.Tail().Height())
	t.Logf("Seed Tail: %v floor, %v", seed.Tail().Height(), seed.Tail().HexHash())

	// create First Receiver
	receiver := testNetwork.NewNode()
	receiver.Start()

	testNetwork.WaitForEstablished()

	runSyncDownload(t, receiver, seed.Tail().BlockData)

	for i := uint64(1); i <= receiver.Tail().Height(); i++ {
		seedTesterBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		receiveTesterBlock, receiveErr := receiver.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedTesterBlock.Hash(), receiveTesterBlock.Hash())
	}
}

func TestForDuplicatedBlock(t *testing.T) {
	const (
		nBlocks              = 10
		indexDuplicatedBlock = 5
	)
	testNetwork := testutil.NewNetwork(t, blockutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist).Block(seed.Tail())
	for i := 1; i < nBlocks; i++ {
		bb = bb.Child().Tx().RandomTx().Execute().SignProposer()
		err := seed.Med.BlockManager().PushBlockData(bb.Build().BlockData)
		require.NoError(t, err)
	}

	require.NoError(t, seed.WaitUntilTailHeight(bb.Build().Height(), 10*time.Second))
	require.Equal(t, uint64(nBlocks), seed.Tail().Height())
	t.Logf("Seed Tail: %v floor, %v", seed.Tail().Height(), seed.Tail().HexHash())

	// create First Receiver
	receiver := testNetwork.NewNode()
	receiver.Start()

	testNetwork.WaitForEstablished()

	duplicatedBlock, err := seed.Med.BlockManager().BlockByHeight(indexDuplicatedBlock)
	require.NoError(t, err)
	require.NoError(t, receiver.Med.BlockManager().PushBlockData(duplicatedBlock.BlockData))

	runSyncDownload(t, receiver, seed.Tail().BlockData)

	for i := uint64(1); i <= receiver.Tail().Height(); i++ {
		seedTesterBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		receiveTesterBlock, receiveErr := receiver.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedTesterBlock.Hash(), receiveTesterBlock.Hash())
	}
}

func TestForkRecover(t *testing.T) {
	const (
		timeSpend       = 600 // 10 minute
		dynastySize     = 21
		nMinorProposers = 6
		nMajorProposers = dynastySize - nMinorProposers
	)
	var (
		blockInterval = int64(dpos.BlockInterval.Seconds())
	)

	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	forkedNode := testNetwork.NewNode()
	forkedNode.Start()

	majorProposers := make(map[common.Address]interface{})
	minorProposers := make(map[common.Address]interface{})
	for _, v := range testNetwork.Seed.Config.Dynasties[0:nMajorProposers] {
		majorProposers[v.Addr] = true
	}
	for _, v := range testNetwork.Seed.Config.Dynasties[nMajorProposers:] {
		minorProposers[v.Addr] = true
	}
	require.Equal(t, nMajorProposers, len(majorProposers))
	require.Equal(t, nMinorProposers, len(minorProposers))

	majorCanonical := make([]*core.BlockData, 0)
	minorCanonical := make([]*core.BlockData, 0)

	init := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist).Block(seed.Tail())
	ts := dpos.NextMintSlot2(time.Now().Unix() - timeSpend)

	bb := init
	for dt := int64(0); dt < timeSpend; dt += blockInterval {
		temp := bb.ChildWithTimestamp(ts + dt).Tx().RandomTx().Execute().SignProposer()
		proposer, err := temp.Build().Proposer()
		require.NoError(t, err)
		if _, ok := majorProposers[proposer]; !ok {
			continue
		}
		bb = temp
		majorCanonical = append(majorCanonical, bb.Build().BlockData)
	}

	bb = init
	for dt := int64(0); dt < timeSpend; dt += blockInterval {
		temp := bb.ChildWithTimestamp(ts + dt).Tx().RandomTx().Execute().SignProposer()
		proposer, err := temp.Build().Proposer()
		require.NoError(t, err)
		if _, ok := minorProposers[proposer]; !ok {
			continue
		}
		bb = temp
		minorCanonical = append(minorCanonical, bb.Build().BlockData)
	}

	for _, bd := range majorCanonical {
		require.NoError(t, seed.Med.BlockManager().PushBlockDataSync(bd, 5*time.Second))
	}
	for _, bd := range minorCanonical {
		require.NoError(t, forkedNode.Med.BlockManager().PushBlockDataSync(bd, 5*time.Second))
	}

	require.NoError(t, seed.WaitUntilTailHeight(majorCanonical[len(majorCanonical)-1].Height(), 10*time.Second))
	require.NoError(t, forkedNode.WaitUntilTailHeight(majorCanonical[len(minorCanonical)-1].Height(), 10*time.Second))

	t.Logf("SeedNode's Tail  : %v floor, %v, %v", seed.Tail().Height(), seed.Tail().HexHash(), seed.Tail().Timestamp())
	t.Logf("ForkedNode's Tail: %v floor, %v, %v", forkedNode.Tail().Height(), forkedNode.Tail().HexHash(), forkedNode.Tail().Timestamp())

	require.False(t, byteutils.Equal(seed.Tail().Hash(), forkedNode.Tail().Hash()))

	cfg := testutil.NewConfig(t)
	cfg.Config.Sync.NumberOfRetries = 256 // testFail probability:(1/2)^256 (same as hash collision probability)
	newbie := testNetwork.NewNodeWithConfig(cfg)
	newbie.Start()

	testNetwork.WaitForEstablished()

	runSyncDownload(t, newbie, forkedNode.Tail().BlockData)
	require.Equal(t, forkedNode.Tail().HexHash(), newbie.Tail().HexHash())

	runSyncDownload(t, newbie, seed.Tail().BlockData)
	require.Equal(t, seed.Tail().HexHash(), newbie.Tail().HexHash())

	newTail := newbie.Tail()
	t.Logf("Height(%v) block of newbie tester	: %v", newTail.Height(), newTail.HexHash())
	t.Logf("Height(%v) block of seed tester	  : %v", newTail.Height(), seed.Tail().HexHash())
	for i := uint64(1); i <= newTail.Height(); i++ {
		seedTesterBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		newbieTesterBlock, receiveErr := newbie.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedTesterBlock.Hash(), newbieTesterBlock.Hash())
	}
}

func TestForAutoActivation1(t *testing.T) {
	const (
		nBlocks              = 100
		syncActivationHeight = uint64(40)
	)

	testNetwork := testutil.NewNetwork(t, 3)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist).Block(seed.Tail())

	// generate blocks on seedTester
	for i := 1; i < nBlocks-1; i++ {
		bb = bb.Child().Tx().RandomTx().Execute().SignProposer()
		require.NoError(t, seed.Med.BlockManager().PushBlockData(bb.Build().BlockData))
	}

	cfg := testutil.NewConfig(t)
	cfg.Config.Sync.SyncActivationHeight = syncActivationHeight
	receiver := testNetwork.NewNodeWithConfig(cfg)
	receiver.Start()

	testNetwork.WaitForEstablished()

	nextMintTs := dpos.NextMintSlot2(time.Now().Unix())
	bb = bb.ChildWithTimestamp(nextMintTs).Tx().RandomTx().Execute().SignProposer()
	bd := bb.Build().BlockData
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bd))

	require.NoError(t, seed.WaitUntilTailHeight(nBlocks, 10*time.Second))

	time.Sleep(time.Unix(bd.Timestamp(), 0).Sub(time.Now()))
	require.NoError(t, seed.Med.BlockManager().BroadCast(seed.Tail().BlockData))

	startTime := time.Now()
	for {
		require.True(t, time.Now().Sub(startTime) < time.Duration(10)*time.Second, "Timeout: Failed to activate sync automatically")
		if receiver.Med.SyncService().IsDownloadActivated() {
			t.Logf("Timespend for auto activate: %v", time.Now().Sub(startTime))
			t.Log("Success to activate sync automatically")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	startTime = time.Now()
	prevSize := uint64(0)
	for receiver.Med.SyncService().IsDownloadActivated() {
		curSize := receiver.Tail().Height()
		require.Truef(t, time.Now().Sub(startTime) < time.Duration(10)*time.Second, "Timeout: sync spend too much time. Current Height(%v)", curSize)
		if curSize > prevSize {
			startTime = time.Now()
		}
		prevSize = curSize

		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("Sync service is unactivated (height:%v)", receiver.Tail().Height())
	require.NoError(t, receiver.WaitUntilTailHeight(nBlocks, 10*time.Second))

	newTail := receiver.Tail()

	t.Logf("Height(%v) block of Seed Node	    (after sync): %v", seed.Tail().Height(), byteutils.Bytes2Hex(seed.Tail().Hash()))
	t.Logf("Height(%v) block of Reciever Node	(after sync): %v", newTail.Height(), byteutils.Bytes2Hex(newTail.Hash()))

	require.Equal(t, seed.Tail().Height(), newTail.Height())

	for i := uint64(1); i <= newTail.Height(); i++ {
		seedNodeBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		receiveNodeBlock, receiveErr := receiver.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedNodeBlock.Hash(), receiveNodeBlock.Hash())
	}
}

func TestForAutoActivation2(t *testing.T) {
	const (
		timeSpend       = 300 // 5 minutes
		dynastySize     = 21
		nMinorProposers = 10
		nMajorProposers = dynastySize - nMinorProposers
	)
	var (
		blockInterval = int64(dpos.BlockInterval.Seconds())
	)

	testNetwork := testutil.NewNetwork(t, dynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	forkedNode := testNetwork.NewNode()
	forkedNode.Start()

	majorProposers := make(map[common.Address]interface{})
	minorProposers := make(map[common.Address]interface{})
	for _, v := range testNetwork.Seed.Config.Dynasties[0:nMajorProposers] {
		majorProposers[v.Addr] = true
	}
	for _, v := range testNetwork.Seed.Config.Dynasties[nMajorProposers:] {
		minorProposers[v.Addr] = true
	}
	require.Equal(t, nMajorProposers, len(majorProposers))
	require.Equal(t, nMinorProposers, len(minorProposers))

	majorCanonical := make([]*core.BlockData, 0)
	minorCanonical := make([]*core.BlockData, 0)

	init := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist).Block(seed.Tail())
	ts := dpos.NextMintSlot2(time.Now().Unix() - timeSpend)

	bb := init
	for dt := int64(0); dt < timeSpend; dt += blockInterval {
		temp := bb.ChildWithTimestamp(ts + dt).Tx().RandomTx().Execute().SignProposer()
		proposer, err := temp.Build().Proposer()
		require.NoError(t, err)
		if _, ok := minorProposers[proposer]; !ok {
			continue
		}
		bb = temp
		minorCanonical = append(minorCanonical, bb.Build().BlockData)
	}

	bb = init
	for dt := int64(0); dt < timeSpend; dt += blockInterval {
		temp := bb.ChildWithTimestamp(ts + dt).Tx().RandomTx().Execute().SignProposer()
		proposer, err := temp.Build().Proposer()
		require.NoError(t, err)
		if _, ok := majorProposers[proposer]; !ok {
			continue
		}
		bb = temp
		majorCanonical = append(majorCanonical, bb.Build().BlockData)
	}

	for _, bd := range majorCanonical {
		require.NoError(t, seed.Med.BlockManager().PushBlockDataSync(bd, 5*time.Second))
	}
	for _, bd := range minorCanonical {
		require.NoError(t, forkedNode.Med.BlockManager().PushBlockDataSync(bd, 5*time.Second))
	}

	require.NoError(t, seed.WaitUntilTailHeight(majorCanonical[len(majorCanonical)-1].Height(), 10*time.Second))
	require.NoError(t, forkedNode.WaitUntilTailHeight(majorCanonical[len(minorCanonical)-1].Height(), 10*time.Second))

	oldTail := forkedNode.Tail()
	t.Logf("Height(%v) block of Reciever Node (before sync): %v", oldTail.Height(), byteutils.Bytes2Hex(oldTail.Hash()))

	nextMintTs := dpos.NextMintSlot2(time.Now().Unix())
	bb = bb.ChildWithTimestamp(nextMintTs).Tx().RandomTx().Execute().SignProposer()
	bd := bb.Build().BlockData
	require.NoError(t, seed.Med.BlockManager().PushBlockData(bd))

	require.NoError(t, seed.WaitUntilTailHeight(bd.Height(), 10*time.Second))

	time.Sleep(time.Unix(bd.Timestamp(), 0).Sub(time.Now()))
	require.NoError(t, seed.Med.BlockManager().BroadCast(seed.Tail().BlockData))

	startTime := time.Now()
	for {
		require.True(t, time.Now().Sub(startTime) < time.Duration(20)*time.Second, "Timeout: Failed to activate sync automatically")
		if forkedNode.Med.SyncService().IsDownloadActivated() {
			t.Logf("Timespend for auto activate: %v", time.Now().Sub(startTime))
			t.Log("Success to activate sync automatically")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	for forkedNode.Med.SyncService().IsDownloadActivated() {
		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("Sync service is unactivated (height:%v)", forkedNode.Tail().Height())
	require.NoError(t, forkedNode.WaitUntilTailHeight(seed.Tail().Height(), 10*time.Second))

	newTail := forkedNode.Tail()

	t.Logf("Height(%v) block of Seed Node	     (after sync): %v", seed.Tail().Height(), byteutils.Bytes2Hex(seed.Tail().Hash()))
	t.Logf("Height(%v) block of Reciever Node	 (after sync): %v", newTail.Height(), byteutils.Bytes2Hex(newTail.Hash()))

	require.Equal(t, seed.Tail().Height(), newTail.Height())

	for i := uint64(1); i <= newTail.Height(); i++ {
		seedNodeBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		receiveNodeBlock, receiveErr := forkedNode.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedNodeBlock.Hash(), receiveNodeBlock.Hash())
	}
}

func runSyncDownload(t *testing.T, node *testutil.Node, target *core.BlockData) {
	ss := node.Med.SyncService()
	require.NoError(t, ss.Download(target))

	startTime := time.Now()
	for ss.IsDownloadActivated() {
		time.Sleep(100 * time.Millisecond)
	}
	assert.Equal(t, target.Height(), node.Tail().Height())
	assert.Equal(t, target.HexHash(), node.Tail().HexHash())
	t.Logf("Sync download complete. height: %v, timeSpend: %v", node.Tail().Height(), time.Now().Sub(startTime))
}
