package sync

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/require"
)

var (
	DOMAIN = "127.0.0.1"
)

type SyncTester struct {
	config       *medletpb.Config
	medService   net.Service
	storage      storage.Storage
	dynasties    test.Dynasties
	blockManager BlockManager
	syncService  *Service
}

func removeRouteTableCache(t *testing.T) {
	_, err := os.Stat("./data.db/routetable.cache")
	if err == nil {
		t.Log("remove routetable.cache")
		os.Remove("./data.db/routetable.cache")
	}
}

func NewSyncTester(t *testing.T, config *medletpb.Config, genesisConf *corepb.Genesis, dynasties test.Dynasties) *SyncTester {
	removeRouteTableCache(t)
	if len(config.Network.Listen) < 1 {
		port := test.FindRandomListenPorts(1)
		addr := fmt.Sprintf("%s:%s", DOMAIN, port[0])
		config.Network.Listen = append(config.Network.Listen, addr)
	}

	ms, err := net.NewMedService(config)
	require.Nil(t, err)

	stor, err := storage.NewMemoryStorage()
	require.Nil(t, err)

	consensus, err := dpos.New(config)
	require.NoError(t, err)

	bm, err := core.NewBlockManager(config)
	require.Nil(t, err)
	bm.Setup(genesisConf, stor, ms, consensus)

	tm := core.NewTransactionManager(config)
	require.NoError(t, err)

	consensus.Setup(genesisConf, bm, tm)

	ss := NewService(config.Sync)
	ss.Setup(ms, bm)

	return &SyncTester{
		config:       config,
		medService:   ms,
		storage:      stor,
		dynasties:    dynasties,
		blockManager: bm,
		syncService:  ss,
	}
}

func (st *SyncTester) Start() {
	st.medService.Start()
	st.syncService.Start()
}

func (st *SyncTester) NodeID() string {
	return st.medService.Node().ID()
}

func TestService_Start(t *testing.T) {
	var (
		nBlocks   = 500
		chunkSize = 20
	)

	//create First Tester(Seed Node)
	genesisConf, dynasties, _ := test.NewTestGenesisConf(t)
	seedTester := NewSyncTester(t, DefaultSyncTesterConfig(), genesisConf, dynasties)
	seedTester.Start()

	for i := 1; i < nBlocks; i++ {
		b := test.NewTestBlock(t, seedTester.blockManager.TailBlock())
		test.SignBlock(t, b, seedTester.dynasties)
		seedTester.blockManager.PushBlockData(b.GetBlockData())
	}
	require.Equal(t, uint64(nBlocks), seedTester.blockManager.TailBlock().Height())
	t.Logf("Seed Tail: %v floor, %v", seedTester.blockManager.TailBlock().Height(), seedTester.blockManager.TailBlock().Hash())

	//create First Receiver
	conf := DefaultSyncTesterConfig()
	seedMultiAddr := convertIpv4ListensToMultiAddrSeeds(seedTester.config.Network.Listen, seedTester.NodeID())
	t.Log(seedMultiAddr)
	conf.Network.Seed = seedMultiAddr
	conf.Sync.DownloadChunkSize = uint64(chunkSize)
	receiveTester := NewSyncTester(t, conf, genesisConf, dynasties)
	receiveTester.Start()

	for {
		if receiveTester.medService.Node().PeersCount() >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	receiveTester.syncService.ActiveDownload()
	for {
		if receiveTester.blockManager.TailBlock().Height() >= uint64(nBlocks-chunkSize+1) {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	newTail := receiveTester.blockManager.TailBlock()
	t.Logf("Height(%v) block of New Node	    : %v", newTail.Height(), newTail.Hash())
	t.Logf("Height(%v) block of Origin Node	: %v", newTail.Height(), seedTester.blockManager.BlockByHeight(newTail.Height()).Hash())
	for i := uint64(1); i <= newTail.Height(); i++ {
		require.NotNil(t, seedTester.blockManager.BlockByHeight(i), "Seeder Height:%v, Hash:%v", i, byteutils.Bytes2Hex(seedTester.blockManager.BlockByHeight(i).Hash().Bytes()))
		require.NotNil(t, receiveTester.blockManager.BlockByHeight(i), "Receiver Height Missing:%v, Hash: %v", i, byteutils.Bytes2Hex(seedTester.blockManager.BlockByHeight(i).Hash().Bytes()))
		require.Equal(t, seedTester.blockManager.BlockByHeight(i).Hash(), receiveTester.blockManager.BlockByHeight(i).Hash())
	}
}

func TestForkResistance(t *testing.T) {
	removeRouteTableCache(t)

	var (
		nMajors   = 6
		nBlocks   = 500
		chunkSize = 20
	)

	//create First Tester(Seed Node)
	genesisConf, dynasties, _ := test.NewTestGenesisConf(t)
	seedTester := NewSyncTester(t, DefaultSyncTesterConfig(), genesisConf, dynasties)
	seedTester.Start()
	t.Log("seedTesterID", seedTester.NodeID())

	//Generate config include seed addresses
	seedMultiAddr := convertIpv4ListensToMultiAddrSeeds(seedTester.config.Network.Listen, seedTester.NodeID())

	//create testers has correct blockchain(Synced with first Tester)
	majorTesters := make([]*SyncTester, nMajors-1)
	for i := 0; i < nMajors-1; i++ {
		conf := DefaultSyncTesterConfig()
		conf.Network.Seed = seedMultiAddr
		conf.Sync.DownloadChunkSize = uint64(chunkSize)

		majorTesters[i] = NewSyncTester(t, conf, genesisConf, dynasties)
		t.Logf("major #%v listen:%v", i, majorTesters[i].config.Network.Listen)
		majorTesters[i].Start()
	}

	//Generate blocks and push to seed tester and major tester
	for i := 1; i < nBlocks; i++ {
		b := test.NewTestBlock(t, seedTester.blockManager.TailBlock())
		test.SignBlock(t, b, seedTester.dynasties)
		seedTester.blockManager.PushBlockData(b.GetBlockData())
		for _, st := range majorTesters {
			st.blockManager.PushBlockData(copyBlockData(t, b.BlockData))
		}
	}

	t.Logf("SeedTester Tail: %v floor, %v", seedTester.blockManager.TailBlock().Height(), seedTester.blockManager.TailBlock().Hash())
	for i, st := range majorTesters {
		t.Logf("MajorTester #%v Tail: %v floor, %v", i, st.blockManager.TailBlock().Height(), st.blockManager.TailBlock().Hash())
		require.Equal(t, seedTester.blockManager.TailBlock().Height(), st.blockManager.TailBlock().Height())
		require.Equal(t, seedTester.blockManager.TailBlock().Hash(), st.blockManager.TailBlock().Hash())
	}

	// create testers has forked blockchain
	nMinors := nMajors - 1
	minorTesters := make([]*SyncTester, nMinors)
	for i := 0; i < nMinors; i++ {
		conf := DefaultSyncTesterConfig()
		conf.Network.Seed = seedMultiAddr
		conf.Sync.DownloadChunkSize = uint64(chunkSize)
		minorTesters[i] = NewSyncTester(t, conf, genesisConf, dynasties)
		t.Logf("minor #%v listen:%v", i, minorTesters[i].config.Network.Listen)
		minorTesters[i].Start()
	}

	//Generate diff blocks and push to minor tester
	for i := 1; i < nBlocks; i++ {
		b := test.NewTestBlock(t, minorTesters[0].blockManager.TailBlock())
		test.SignBlock(t, b, seedTester.dynasties)

		for _, st := range minorTesters {
			st.blockManager.PushBlockData(copyBlockData(t, b.BlockData))
		}
	}

	for i, st := range minorTesters {
		t.Logf("MinorTester #%v Tail: %v floor, %v", i, st.blockManager.TailBlock().Height(), st.blockManager.TailBlock().Hash())
		require.Equal(t, minorTesters[0].blockManager.TailBlock().Height(), st.blockManager.TailBlock().Height())
		require.Equal(t, minorTesters[0].blockManager.TailBlock().Hash(), st.blockManager.TailBlock().Hash())
	}

	conf := DefaultSyncTesterConfig()
	conf.Network.Seed = seedMultiAddr
	conf.Sync.DownloadChunkSize = uint64(chunkSize)
	newbieTester := NewSyncTester(t, conf, genesisConf, dynasties)
	newbieTester.Start()

	for {
		nPeers := newbieTester.medService.Node().PeersCount()
		if nPeers >= int32(nMajors+nMinors) {
			t.Log("Number of connected peers:", nPeers)
			break
		}
		time.Sleep(2 * time.Second)
	}

	newbieTester.syncService.ActiveDownload()

	for {
		if newbieTester.blockManager.TailBlock().Height() > uint64(nBlocks-chunkSize+1) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	newTail := newbieTester.blockManager.TailBlock()
	t.Logf("Height(%v) block of newbie tester	: %v", newTail.Height(), newTail.Hash())
	t.Logf("Height(%v) block of seed tester	  : %v", newTail.Height(), seedTester.blockManager.BlockByHeight(newTail.Height()).Hash())
	for i := uint64(1); i <= newTail.Height(); i++ {
		require.Equal(t, seedTester.blockManager.BlockByHeight(i).Hash(), newbieTester.blockManager.BlockByHeight(i).Hash())
	}
	//t.Logf(newbieTester.syncService.Download.finishedTasks)
}

func DefaultSyncTesterConfig() *medletpb.Config {
	return &medletpb.Config{
		Global: &medletpb.GlobalConfig{
			ChainId: 1,
			Datadir: "data.db",
		},
		Network: &medletpb.NetworkConfig{
			Seed:                       make([]string, 0),
			Listen:                     make([]string, 0),
			PrivateKey:                 "",
			NetworkId:                  0,
			RouteTableSyncLoopInterval: 2000,
		},
		Chain: &medletpb.ChainConfig{
			Coinbase: "02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c",
			Miner:    "02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c",
		},
		Sync: &medletpb.SyncConfig{
			SeedingMinChunkSize:        1,
			SeedingMaxChunkSize:        100,
			SeedingMaxConcurrentPeers:  5,
			DownloadChunkSize:          50,
			DownloadMaxConcurrentTasks: 5,
			DownloadChunkCacheSize:     10,
		},
	}
}

func convertIpv4ListensToMultiAddrSeeds(ipv4s []string, nodeID string) (multiAddrs []string) {
	for _, ipv4 := range ipv4s {
		splitIpv4 := strings.Split(ipv4, ":")
		mAddr := fmt.Sprintf(fmt.Sprintf(
			"/ip4/%v/tcp/%v/ipfs/%s",
			splitIpv4[0],
			splitIpv4[1],
			nodeID,
		))
		multiAddrs = append(multiAddrs, mAddr)
	}

	return multiAddrs
}

func copyBlockData(t *testing.T, origin *core.BlockData) *core.BlockData {
	pbBlock, err := origin.ToProto()
	require.Nil(t, err)
	copiedBlockData := new(core.BlockData)
	copiedBlockData.FromProto(pbBlock)
	return copiedBlockData
}
