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
	"context"
	"io/ioutil"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	netpb "github.com/medibloc/go-medibloc/net/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

// BootstrapConfig is a struct for bootstrapping node
type BootstrapConfig struct {
	MinConnThreshold  uint32
	Period            time.Duration
	ConnectionTimeout time.Duration
	BootstrapPeers    []pstore.PeerInfo // seed host
}

// NewBootstrapConfig return new BootstrapConfig
func NewBootstrapConfig(cfg *medletpb.Config) (bConfig *BootstrapConfig, err error) {
	bootstrapPeriod := DefaultBootstrapPeriod
	if cfg.Network.BootstrapPeriod != 0 {
		bootstrapPeriod = time.Duration(cfg.Network.BootstrapPeriod) * time.Second
	}

	bConfig = &BootstrapConfig{
		MinConnThreshold:  cfg.Network.MinimumConnections,
		Period:            bootstrapPeriod,
		ConnectionTimeout: 3 * time.Second,
		BootstrapPeers:    make([]pstore.PeerInfo, 0),
	}

	for _, seed := range cfg.Network.Seeds {
		pi, err := peerInfoFromProto(seed)
		if err != nil {
			return nil, err
		}
		bConfig.BootstrapPeers = append(bConfig.BootstrapPeers, pi)
	}

	return bConfig, nil
}

// Bootstrap run bootstrap
func (node *Node) Bootstrap() {
	cfg := node.bootstrapConfig
	connected := node.Network().Peers()
	if len(connected) >= int(cfg.MinConnThreshold) {
		return
	}
	node.DHTSync()
	bootstrapConnect(node.Host, cfg.BootstrapPeers)
}

func bootstrapConnect(ph host.Host, peers []pstore.PeerInfo) {
	logging.Console().WithFields(logrus.Fields{
		"current_connections": len(ph.Network().Conns()),
	}).Info("Start bootstrap connect")

	var wg sync.WaitGroup
	for _, p := range peers {
		if p.ID == ph.ID() {
			continue
		}
		wg.Add(1)
		go func(p pstore.PeerInfo) {
			defer wg.Done()

			if ph.Network().Connectedness(p.ID) == net.Connected {
				return
			}
			logging.Console().WithFields(logrus.Fields{
				"to":   p.ID.Pretty(),
				"from": ph.ID().Pretty(),
			}).Debug("bootstrap connection start")

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)
			if err := ph.Connect(context.Background(), p); err != nil {
				logging.Console().WithFields(logrus.Fields{
					"to":   p.ID.Pretty(),
					"from": ph.ID().Pretty(),
				}).Info("bootstrap connection failed")
				return
			}
			logging.Console().WithFields(logrus.Fields{
				"to":   p.ID.Pretty(),
				"from": ph.ID().Pretty(),
			}).Info("bootstrap connection succeed")
		}(p)
	}
	wg.Wait()

	logging.Console().WithFields(logrus.Fields{
		"current_connections": len(ph.Network().Conns()),
	}).Info("finishing bootstrap connect")
}

// SaveCache save host's peerstore to cache file
func (node *Node) SaveCache() {
	savePeerStoreToCache(node.Peerstore(), node.cacheFile)
}

// savePeerStoreToCache save peerstore to cache file
func savePeerStoreToCache(ps pstore.Peerstore, cacheFile string) {
	pbPeers := new(netpb.Peers)
	for _, id := range ps.Peers() {
		p := ps.PeerInfo(id)
		pbPeer := PeerInfoToProto(p)
		pbPeers.Peers = append(pbPeers.Peers, pbPeer)
	}

	str := proto.MarshalTextString(pbPeers)

	err := ioutil.WriteFile(cacheFile, []byte(str), 0644)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"cacheFile": cacheFile,
			"err":       err,
		}).Warn("failed to save peers to cache file")
		return
	}
}

func peerInfoFromProto(pb *netpb.PeerInfo) (pstore.PeerInfo, error) {
	pid, err := peer.IDB58Decode(pb.Id)
	if err != nil {
		return pstore.PeerInfo{}, err
	}
	addrs := make([]multiaddr.Multiaddr, len(pb.Addrs))
	for i, addr := range pb.Addrs {
		addrs[i], err = multiaddr.NewMultiaddr(addr)
		if err != nil {
			return pstore.PeerInfo{}, err
		}
	}
	return pstore.PeerInfo{
		ID:    pid,
		Addrs: addrs,
	}, nil
}
