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

package net

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/multiformats/go-multiaddr"
)

// const
const ( // TODO delete redundant vars
	DefaultBucketCapacity               = 64
	DefaultRoutingTableMaxLatency       = 10
	DefaultPrivateKeyPath               = "conf/network.key"
	DefaultPrivateKeyCachePath          = "data/network.key"
	DefaultMaxSyncNodes                 = 64
	DefaultChainID                      = 1
	DefaultRoutingTableDir              = ""
	DefaultRouteTableSyncLoopInterval   = 30 * time.Millisecond
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
	PrivateKeyCachePath          string
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
func NewP2PConfig(cfg *medletpb.Config) *Config {

	globalConf := cfg.Global
	networkConf := cfg.Network
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
	config.PrivateKeyCachePath = fmt.Sprintf("%v/network.key", cfg.Global.Datadir)
	// Chain ID.
	config.ChainID = globalConf.ChainId

	// routing table dir.
	// TODO: health check for peers in routing cache file
	if checkPathConfig(globalConf.Datadir) == false {
		if err := os.MkdirAll(globalConf.Datadir, 0700); err != nil {
			panic("Failed to create logger folder:" + globalConf.Datadir + ". err:" + err.Error())
		}
	}
	config.RoutingTableDir = globalConf.Datadir

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

	// set route table syncing interval from config
	routeTableSyncLoopInterval := time.Duration(networkConf.RouteTableSyncLoopInterval) * time.Millisecond
	if routeTableSyncLoopInterval == 0 {
		routeTableSyncLoopInterval = DefaultRouteTableSyncLoopInterval
	}
	config.RouteTableSyncLoopInterval = routeTableSyncLoopInterval

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
		DefaultPrivateKeyCachePath,
		DefaultListen,
		DefaultMaxSyncNodes,
		DefaultChainID,
		DefaultRoutingTableDir,
		DefaultRouteTableSyncLoopInterval,
		DefaultRouteTableSaveToDiskInterval,
	}
}
