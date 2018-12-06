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

	"github.com/medibloc/go-medibloc/sync"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/net"
	syncpb "github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Start(t *testing.T) {
	var (
		nBlocks   = 100
		chunkSize = 20
	)

	testNetwork := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()
	for i := 1; i < nBlocks; i++ {
		var b *core.Block
		if i == 1 {
			b = bb.Block(tail).Child().Stake().Tx().RandomTx().Execute().SignProposer().Build()
		} else {
			b = bb.Block(tail).Child().Tx().RandomTx().Execute().SignProposer().Build()
		}
		tail = b
		err := seed.Med.BlockManager().PushBlockDataSync(b.BlockData)
		require.NoError(t, err)
	}

	require.Equal(t, uint64(nBlocks), seed.Tail().Height())
	t.Logf("Seed Tail: %v floor, %v", seed.Tail().Height(), seed.Tail().Hash())

	//create First Receiver

	receiver := testNetwork.NewNode()
	receiver.Start()

	receiver.Med.SyncService().ActiveDownload(seed.Tail().Height())
	count := 0
	prevSize := uint64(0)
	for {
		if !receiver.Med.SyncService().IsDownloadActivated() {
			break
		}

		curSize := receiver.Tail().Height()
		if curSize == prevSize {
			count++
		} else {
			count = 0
		}
		if count > 100 {
			t.Logf("Timeout for syncTest: Current Height(%v)", curSize)
			require.True(t, false)
		}
		prevSize = curSize

		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("receiver Tail: %v floor, %v", receiver.Tail().Height(), receiver.Tail().Hash())
	require.True(t, int(receiver.Tail().Height()) > nBlocks-chunkSize)
	for i := uint64(1); i <= receiver.Tail().Height(); i++ {
		seedTesterBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		receiveTesterBlock, receiveErr := receiver.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedTesterBlock.Hash(), receiveTesterBlock.Hash())
	}
}

func TestForkResistance(t *testing.T) {
	var (
		nMajors   = 6
		nBlocks   = 50
		chunkSize = 20
	)

	testNetwork := testutil.NewNetwork(t, 3)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	majorNodes := make([]*testutil.Node, nMajors-1)
	for i := 0; i < nMajors-1; i++ {
		majorNodes[i] = testNetwork.NewNode()
		majorNodes[i].Start()
	}

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist)

	//Generate blocks and push to seed tester and major tester

	for i := 1; i < nBlocks; i++ {
		tail := seed.Tail()
		var b *core.Block
		if i == 1 {
			b = bb.Block(tail).Child().Stake().Tx().RandomTx().Execute().SignProposer().Build()
		} else {
			b = bb.Block(tail).Child().Tx().RandomTx().Execute().SignProposer().Build()
		}

		err := seed.Med.BlockManager().PushBlockDataSync(b.BlockData)
		assert.NoError(t, err)

		for _, n := range majorNodes {
			b, err := b.Clone()
			assert.NoError(t, err)
			err = n.Med.BlockManager().PushBlockDataSync(b.BlockData)
			assert.NoError(t, err)
		}
	}

	t.Logf("SeedTester Tail: %v floor, %v", seed.Tail().Height(), seed.Tail().Hash())
	for i, n := range majorNodes {
		t.Logf("MajorNode #%v Tail: %v floor, %v", i, n.Tail().Height(), n.Tail().Hash())
		require.Equal(t, seed.Tail().Height(), n.Tail().Height())
		require.Equal(t, seed.Tail().Hash(), n.Tail().Hash())
	}

	// create testers has forked blockchain
	nMinors := nMajors - 1
	minorNodes := make([]*testutil.Node, nMinors)
	for i := 0; i < nMinors; i++ {
		minorNodes[i] = testNetwork.NewNode()
		minorNodes[i].Start()
	}

	//Generate diff blocks and push to minor tester
	for i := 1; i < nBlocks; i++ {
		tail := minorNodes[0].Tail()
		var b *core.Block
		if i == 1 {
			b = bb.Block(tail).Child().Stake().Tx().RandomTx().Execute().SignProposer().Build()
		} else {
			b = bb.Block(tail).Child().Tx().RandomTx().Execute().SignProposer().Build()
		}

		for _, n := range minorNodes {
			b, err := b.Clone()
			assert.NoError(t, err)
			err = n.Med.BlockManager().PushBlockDataSync(b.BlockData)
			assert.NoError(t, err)
		}
	}

	for i, n := range minorNodes {
		t.Logf("MinorNode #%v Tail: %v floor, %v", i, n.Tail().Height(), n.Tail().Hash())
		require.Equal(t, minorNodes[0].Tail().Height(), n.Tail().Height())
		require.Equal(t, minorNodes[0].Tail().Hash(), n.Tail().Hash())
	}

	require.False(t, byteutils.Equal(minorNodes[0].Tail().Hash(), majorNodes[0].Tail().Hash()))

	cfg := testutil.NewConfig(t)
	cfg.Config.Sync.DownloadChunkCacheSize = uint64(chunkSize)
	newbie := testNetwork.NewNodeWithConfig(cfg)

	newbie.Start()

	testNetwork.WaitForEstablished()

	newbie.Med.SyncService().ActiveDownload(seed.Tail().Height())

	count := 0
	prevSize := uint64(0)
	for {
		if !newbie.Med.SyncService().IsDownloadActivated() {
			break
		}

		curSize := newbie.Tail().Height()
		if curSize == prevSize {
			count++
		} else {
			count = 0
		}
		if count > 1000 {
			t.Logf("Current Height(%v)", curSize)
			require.True(t, false)
		}
		prevSize = curSize

		time.Sleep(100 * time.Millisecond)
	}

	newTail := newbie.Tail()
	t.Logf("Height(%v) block of newbie tester	: %v", newTail.Height(), newTail.Hash())
	t.Logf("Height(%v) block of seed tester	  : %v", newTail.Height(), seed.Tail().Hash())
	for i := uint64(1); i <= newTail.Height(); i++ {
		seedTesterBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		newbieTesterBlock, receiveErr := newbie.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedTesterBlock.Hash(), newbieTesterBlock.Hash())
	}
}

func TestForAutoActivation(t *testing.T) {
	var (
		nBlocks              = 119
		chunkSize            = 20
		syncActivationHeight = uint64(40)
		//nBackward            = 2
	)

	testNetwork := testutil.NewNetwork(t, 3)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()
	t.Log("Seed ")

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist)
	tail := seed.Tail()

	// generate blocks (height:2~nBlocks-1) on seedTester
	for i := 1; i < nBlocks-1; i++ {
		var b *core.Block
		if i == 1 {
			b = bb.Block(tail).Child().Stake().Tx().RandomTx().Execute().SignProposer().Build()
		} else {
			b = bb.Block(tail).Child().Tx().RandomTx().Execute().SignProposer().Build()
		}
		tail = b
		err := seed.Med.BlockManager().PushBlockDataSync(b.BlockData)
		require.NoError(t, err)
	}
	require.Equal(t, nBlocks-1, int(seed.Tail().Height()))

	cfg := testutil.NewConfig(t)
	cfg.Config.Sync.SyncActivationHeight = syncActivationHeight
	receiver := testNetwork.NewNodeWithConfig(cfg)
	receiver.Start()

	testNetwork.WaitForEstablished()

	nextMintTs := dpos.NextMintSlot2(time.Now().Unix())
	b := bb.Block(seed.Tail()).ChildWithTimestamp(nextMintTs).Tx().RandomTx().Execute().SignProposer().Build()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(b.BlockData))

	time.Sleep(time.Unix(b.Timestamp(), 0).Sub(time.Now()))

	startTime := time.Now()
	for {
		require.True(t, time.Now().Sub(startTime) < time.Duration(10)*time.Second, "Timeout: Failed to activate sync automatically")
		seed.Med.BlockManager().BroadCast(seed.Tail().BlockData)
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

	newTail := receiver.Tail()

	t.Logf("Height(%v) block of Seed Node	    (after sync): %v", seed.Tail().Height(), byteutils.Bytes2Hex(seed.Tail().Hash()))
	t.Logf("Height(%v) block of Reciever Node	(after sync): %v", newTail.Height(), byteutils.Bytes2Hex(newTail.Hash()))

	require.True(t, newTail.Height() <= seed.Tail().Height(), "Receiver height is too high")
	require.True(t, int(newTail.Height()) > nBlocks-chunkSize, "Receiver height is too low")

	nextMintTs = dpos.NextMintSlot2(time.Now().Unix())
	b = bb.Block(seed.Tail()).ChildWithTimestamp(nextMintTs).Tx().RandomTx().Execute().SignProposer().Build()
	require.NoError(t, seed.Med.BlockManager().PushBlockData(b.BlockData))
	time.Sleep(time.Unix(b.Timestamp(), 0).Sub(time.Now()))
	seed.Med.BlockManager().BroadCast(b.BlockData)

	startTime = time.Now()
	prevSize = receiver.Tail().Height()
	for receiver.Tail().Height() < seed.Tail().Height() {

		curSize := receiver.Tail().Height()
		require.Truef(t, time.Now().Sub(startTime) < time.Duration(10)*time.Second, "Timeout: request parent block(%v)", curSize)

		if curSize > prevSize {
			startTime = time.Now()
		}
		prevSize = curSize

		time.Sleep(100 * time.Millisecond)
	}

	newTail = receiver.Tail()
	t.Logf("Height(%v) block of Seed Node	    (after next mintblock): %v", seed.Tail().Height(), byteutils.Bytes2Hex(seed.Tail().Hash()))
	t.Logf("Height(%v) block of Reciever Node	(after next mintblock): %v", newTail.Height(), byteutils.Bytes2Hex(newTail.Hash()))

	for i := uint64(1); i <= newTail.Height(); i++ {
		seedNodeBlock, seedErr := seed.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, seedErr, " Missing Seeder Height:%v", i)
		receiveNodeBlock, receiveErr := receiver.Med.BlockManager().BlockByHeight(i)
		require.Nil(t, receiveErr, "Missing Receiver Height :%v", i)

		require.Equal(t, seedNodeBlock.Hash(), receiveNodeBlock.Hash())
	}
}

func TestForInvalidMessageToSeed(t *testing.T) {
	var (
		nBlocks   = 5
		chunkSize = 2
	)

	testNetwork := testutil.NewNetwork(t, 3)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	bb := blockutil.New(t, testNetwork.DynastySize).AddKeyPairs(seed.Config.TokenDist)

	seedingMinChunkSize := seed.Config.Config.Sync.SeedingMinChunkSize
	seedingMaxChunkSize := seed.Config.Config.Sync.SeedingMaxChunkSize

	tail := seed.Tail()
	for i := 1; i < nBlocks; i++ {
		var b *core.Block
		if i == 1 {
			b = bb.Block(tail).Child().Stake().Tx().RandomTx().Execute().SignProposer().Build()
		} else {
			b = bb.Block(tail).Child().Tx().RandomTx().Execute().SignProposer().Build()
		}
		tail = b
		err := seed.Med.BlockManager().PushBlockDataSync(b.BlockData)
		require.NoError(t, err)
	}
	require.Equal(t, uint64(nBlocks), seed.Tail().Height())
	t.Logf("Seed Tail: %v floor, %v", seed.Tail().Height(), seed.Tail().Hash())

	//create First Receiver

	cfg := testutil.NewConfig(t)
	cfg.Config.Sync.DownloadChunkSize = uint64(chunkSize)
	receiver := testNetwork.NewNode()
	receiver.Start()

	testNetwork.WaitForEstablished()

	// Error on wrong height
	mq := new(syncpb.MetaQuery)
	mq.From = uint64(nBlocks + 1)
	mq.Hash = receiver.Tail().Hash()
	mq.ChunkSize = 10

	sendData, err := proto.Marshal(mq)
	require.NoError(t, err)
	receiver.Med.NetService().SendMessageToPeers(sync.SyncMetaRequest, sendData, net.MessagePriorityLow, new(net.ChainSyncPeersFilter))

	// Error on wrong hash
	mq = new(syncpb.MetaQuery)
	mq.From = uint64(nBlocks)
	mq.Hash = receiver.Tail().Hash()
	mq.ChunkSize = 10

	sendData, err = proto.Marshal(mq)
	require.NoError(t, err)
	receiver.Med.NetService().SendMessageToPeers(sync.SyncMetaRequest, sendData, net.MessagePriorityLow, new(net.ChainSyncPeersFilter))

	// Error on chunksize check
	mq = new(syncpb.MetaQuery)
	mq.From = receiver.Tail().Height()
	mq.Hash = receiver.Tail().Hash()
	mq.ChunkSize = seedingMaxChunkSize + 1

	sendData, err = proto.Marshal(mq)
	require.NoError(t, err)
	receiver.Med.NetService().SendMessageToPeers(sync.SyncMetaRequest, sendData, net.MessagePriorityLow, new(net.ChainSyncPeersFilter))

	mq = new(syncpb.MetaQuery)
	mq.From = receiver.Tail().Height()
	mq.Hash = receiver.Tail().Hash()
	mq.ChunkSize = seedingMinChunkSize - 1

	sendData, err = proto.Marshal(mq)
	require.NoError(t, err)
	receiver.Med.NetService().SendMessageToPeers(sync.SyncMetaRequest, sendData, net.MessagePriorityLow, new(net.ChainSyncPeersFilter))

	// Too close to make one chunk
	mq = new(syncpb.MetaQuery)
	mq.From = uint64(1)
	mq.Hash = receiver.Tail().Hash()
	mq.ChunkSize = uint64(seed.Tail().Height() + 1)

	sendData, err = proto.Marshal(mq)
	require.NoError(t, err)
	receiver.Med.NetService().SendMessageToPeers(sync.SyncMetaRequest, sendData, net.MessagePriorityLow, new(net.ChainSyncPeersFilter))

	cq := new(syncpb.BlockChunkQuery)
	cq.From = receiver.Tail().Height()
	cq.ChunkSize = seedingMaxChunkSize + 1

	sendData, err = proto.Marshal(cq)
	require.NoError(t, err)
	receiver.Med.NetService().SendMessageToPeer(sync.SyncBlockChunkRequest, sendData, net.MessagePriorityLow, seed.Med.NetService().Node().ID().Pretty())

	// From + chunk size > tail height
	cq = new(syncpb.BlockChunkQuery)
	cq.From = uint64(1)
	cq.ChunkSize = uint64(seed.Tail().Height() + 1)

	sendData, err = proto.Marshal(cq)
	require.NoError(t, err)
	receiver.Med.NetService().SendMessageToPeer(sync.SyncBlockChunkRequest, sendData, net.MessagePriorityLow, seed.Med.NetService().Node().ID().Pretty())

	time.Sleep(3 * time.Second)
	require.True(t, seed.Med.SyncService().IsSeedActivated(), "SeedTester is shutdown")
}
func TestForUnmarshalFailedMsg(t *testing.T) {

	//create First Tester(Seed Node)
	testNetwork := testutil.NewNetwork(t, 3)
	defer testNetwork.Cleanup()

	seed := testNetwork.NewSeedNode()
	seed.Start()

	//create Abuse Tester
	nTesters := 4
	abuseNodes := make([]*testutil.Node, nTesters)
	for i := 0; i < nTesters; i++ {
		abuseNodes[i] = testNetwork.NewNode()
		t.Logf("Tester #%v listen:%v", i, abuseNodes[i].Config.Config.Network.Listens)
		abuseNodes[i].Start()
	}

	testNetwork.WaitForEstablished()

	seedID := seed.Med.NetService().Node().ID()

	// Error on unmarshal
	dummyData := []byte{72, 101, 108, 108, 111, 44, 32, 119, 111, 114, 108, 100}
	abuseNodes[0].Med.NetService().SendMessageToPeer(sync.SyncMetaRequest, dummyData, net.MessagePriorityLow, seedID.Pretty())
	abuseNodes[1].Med.NetService().SendMessageToPeer(sync.SyncBlockChunkRequest, dummyData, net.MessagePriorityLow, seedID.Pretty())

	seed.Med.SyncService().ActiveDownload(100)
	abuseNodes[2].Med.NetService().SendMessageToPeer(sync.SyncMeta, dummyData, net.MessagePriorityLow, seedID.Pretty())
	abuseNodes[3].Med.NetService().SendMessageToPeer(sync.SyncBlockChunk, dummyData, net.MessagePriorityLow, seedID.Pretty())

	count := 0
	for {
		nPeers := seed.Med.NetService().Node().EstablishedPeersCount()
		if nPeers == int32(0) {
			t.Logf("Disconnection complete.(Connected Node: %v)", nPeers)
			break
		}
		require.Truef(t, count < 100, "Timeout(Connected nNode: %v)", nPeers)
		count++
		time.Sleep(100 * time.Millisecond)
	}
}
