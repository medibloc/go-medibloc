package net

import (
	"fmt"
	"net"
	"time"

	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/multiformats/go-multiaddr"
)

// const
const ( // TODO delete redundant vars
	DefaultBucketCapacity               = 64
	DefaultRoutingTableMaxLatency       = 10
	DefaultPrivateKeyPath               = "conf/network.key"
	DefaultMaxSyncNodes                 = 64
	DefaultChainID                      = 1
	DefaultRoutingTableDir              = ""
	DefaultRouteTableSyncLoopInterval   = 30 * time.Second
	DefaultRouteTableSaveToDiskInterval = 3 * 60 * time.Second
)

// Default Configuration in P2P network
var (
	DefaultListen = []string{"0.0.0.0:8680"}

	RouteTableCacheFileName = "routetable.cache"

	MaxPeersCountForSyncResp = 32
)

// Config TODO: move to proto config.
type Config struct {
	Bucketsize                   int
	Latency                      time.Duration
	BootNodes                    []multiaddr.Multiaddr
	PrivateKeyPath               string
	Listen                       []string
	MaxSyncNodes                 int
	ChainID                      uint32
	RoutingTableDir              string
	RouteTableSyncLoopInterval   time.Duration
	RouteTableSaveToDiskInterval time.Duration
}

// Medlet interface breaks cycle import dependency.
type Medlet interface {
	Config() *medletpb.Config
}

// NewP2PConfig return new config object.
func NewP2PConfig(m Medlet) *Config {
	chainConf := m.Config().Chain
	networkConf := m.Config().Network
	config := NewConfigFromDefaults()

	// listen.
	if len(networkConf.Listen) == 0 {
		panic("Missing network.listen config.")
	}
	if err := verifyListenAddress(networkConf.Listen); err != nil {
		panic(fmt.Sprintf("Invalid network.listen config: err is %s, config value is %s.", err, networkConf.Listen))
	}
	config.Listen = networkConf.Listen

	// private key path.
	if checkPathConfig(networkConf.PrivateKey) == false {
		panic(fmt.Sprintf("The network private key path %s is not exist.", networkConf.PrivateKey))
	}
	config.PrivateKeyPath = networkConf.PrivateKey

	// Chain ID.
	config.ChainID = chainConf.ChainId

	// routing table dir.
	// TODO: using diff dir for temp files.
	if checkPathConfig(chainConf.Datadir) == false {
		panic(fmt.Sprintf("The chain data directory %s is not exist.", chainConf.Datadir))
	}
	config.RoutingTableDir = chainConf.Datadir

	// seed server address.
	seeds := networkConf.Seed
	if len(seeds) > 0 {
		config.BootNodes = make([]multiaddr.Multiaddr, len(seeds))
		for i, v := range seeds {
			addr, err := multiaddr.NewMultiaddr(v)
			if err != nil {
				panic(fmt.Sprintf("Invalid seed address config: err is %s, config value is %s.", err, v))
			}
			config.BootNodes[i] = addr
		}
	}

	return config
}

func localHost() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

// NewConfigFromDefaults return new config from defaults.
func NewConfigFromDefaults() *Config {
	return &Config{
		DefaultBucketCapacity,
		DefaultRoutingTableMaxLatency,
		[]multiaddr.Multiaddr{},
		DefaultPrivateKeyPath,
		DefaultListen,
		DefaultMaxSyncNodes,
		DefaultChainID,
		DefaultRoutingTableDir,
		DefaultRouteTableSyncLoopInterval,
		DefaultRouteTableSaveToDiskInterval,
	}
}
