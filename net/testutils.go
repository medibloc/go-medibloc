package net

import (
	"math/rand"
	"strconv"

	"time"

	"fmt"

	"sync"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/medibloc/go-medibloc/util/logging"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

// const for test
const (
	TestBucketCapacity               = 8
	TestRoutingTableMaxLatency       = 2
	TestMaxSyncNodes                 = 8
	TestChainID                      = 123
	TestRoutingTableDir              = "./testdata/cache/"
	TestRouteTableSyncLoopInterval   = 100 * time.Millisecond
	TestRouteTableSaveToDiskInterval = 1 * time.Second
	RouteTableCheckInterval          = 100 * time.Millisecond
)

func makeNewTestNode(privateKeyPath string) (*Node, error) {
	config := makeNewTestP2PConfig(privateKeyPath)
	node, err := NewNode(config)
	if err != nil {
		return nil, err
	}
	node.routeTable.cacheFilePath += fmt.Sprintf(".%s", node.ID())

	return node, err
}

// makeAndSetNewTestNodes returns nodes(seed nodes first) and its IDs with seedNodes configuration
func makeAndSetNewTestNodes(nodeNum int, seedNum int) ([]*Node, []peer.ID, error) {
	nodeArr := make([]*Node, nodeNum)
	var seedNodes []ma.Multiaddr
	var seedNodeIDs, allNodeIDs []peer.ID
	var err error

	// make all test nodes
	for i := 0; i < nodeNum; i++ {
		nodeArr[i], err = makeNewTestNode("")
		if err != nil {
			return nil, nil, err
		}
	}

	// set value of seedNodes, seedNodeIDs
	for i := 0; i < seedNum; i++ {
		seedMultiaddrs, err := convertListenAddrToMultiAddr(nodeArr[i].config.Listen)
		if err != nil {
			return nil, nil, err
		}
		newSeedNodes, err := convertMultiAddrToIPFSMultiAddr(seedMultiaddrs, nodeArr[i].ID())
		if err != nil {
			return nil, nil, err
		}
		for _, v := range newSeedNodes {
			seedNodes = append(seedNodes, v)
		}
		seedNodeIDs = append(seedNodeIDs, nodeArr[i].id)
	}

	// set value of allNodeIDs
	for i := 0; i < nodeNum; i++ {
		allNodeIDs = append(allNodeIDs, nodeArr[i].id)
	}

	// setup seedNodes to every nodes
	for i := 0; i < nodeNum; i++ {
		nodeArr[i].routeTable.seedNodes = seedNodes
	}

	return nodeArr, allNodeIDs, nil
}

func makeNewTestP2PConfig(privateKeyPath string) *Config {

	randomListen := makeRandomListen(10000, 20000)
	if err := verifyListenAddress(randomListen); err != nil {
		panic(fmt.Sprintf("Invalid random listen config: err is %s, config value is %s.", err, randomListen))
	}

	// private key path.
	if checkPathConfig(privateKeyPath) == false {
		panic(fmt.Sprintf("The network private key path %s is not exist.", privateKeyPath))
	}

	config := &Config{
		TestBucketCapacity,
		TestRoutingTableMaxLatency,
		[]ma.Multiaddr{},
		privateKeyPath,
		randomListen,
		TestMaxSyncNodes,
		TestChainID,
		TestRoutingTableDir,
		TestRouteTableSyncLoopInterval,
		TestRouteTableSaveToDiskInterval,
	}

	return config
}

// TODO: find random free port
func makeRandomListen(low int64, high int64) []string {
	if low > high {
		return []string{}
	}

	rand.Seed(time.Now().UnixNano())
	randomPort := rand.Int63()%(high-low) + low
	randomPortString := strconv.FormatInt(randomPort, 10)
	//randomListen := []string{"0.0.0.0:" + randomPortString}
	randomListen := []string{"localhost:" + randomPortString}

	return randomListen
}

func waitRouteTableSyncLoop(wg *sync.WaitGroup, node *Node, nodeIDs []peer.ID) {
	defer wg.Done()
	ids := make([]peer.ID, len(nodeIDs))
	copy(ids, nodeIDs)
	remainIteration := 100

	for len(ids) > 0 {
		i := 0
		for _, v := range ids {
			if node.routeTable.peerStore.Addrs(v) == nil {
				ids[i] = v
				i++
			}
		}
		ids = ids[:i]

		time.Sleep(RouteTableCheckInterval)

		remainIteration--
		// fail if (sec * len(nodeIDs)) seconds left
		if remainIteration < 1 && len(ids) > 0 {
			logging.Console().WithFields(logrus.Fields{
				"node ID":                          node.ID(),
				"routeTable not synced node count": len(ids),
			}).Warn("route table not synced in time")
			return
		}
	}
}

// PrintRouteTablePeers prints peers in route table
func PrintRouteTablePeers(table *RouteTable) {
	logging.Console().WithFields(logrus.Fields{
		"peerCount": len(table.peerStore.Peers()),
	}).Info(fmt.Sprintf("routeTable peer count of nodeID: %s", table.node.ID()))

	for idx, p := range table.peerStore.Peers() {
		logging.Console().WithFields(logrus.Fields{
			"peerID": p.Pretty(),
		}).Info(fmt.Sprintf("routeTable peer of index %d", idx))
	}
}
