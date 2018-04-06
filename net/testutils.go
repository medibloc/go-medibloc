package net

import (
	"math/rand"
	"strconv"

	"time"

	"fmt"

	"github.com/multiformats/go-multiaddr"
)

// const for test
const (
	TestBucketCapacity               = 8
	TestRoutingTableMaxLatency       = 2
	TestMaxSyncNodes                 = 8
	TestChainID                      = 123
	TestRoutingTableDir              = "./testdata/cache/"
	TestRouteTableSyncLoopInterval   = 300 * time.Millisecond
	TestRouteTableSaveToDiskInterval = 1 * time.Second
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
		[]multiaddr.Multiaddr{},
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
