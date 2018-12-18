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
	"os"
	"path"
	"time"

	"github.com/gogo/protobuf/proto"
	libp2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	netpb "github.com/medibloc/go-medibloc/net/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

// Node the host can be used as both the client and the server
type Node struct {
	host.Host
	context context.Context

	dht       *dht.IpfsDHT
	dhtTicker chan time.Time

	messageSender   *messageSender
	messageReceiver *messageReceiver

	chainID         uint32
	bloomFilter     *BloomFilter
	bootstrapConfig *BootstrapConfig
	cacheFile       string
	cachePeriod     time.Duration
}

//Addrs returns opened addrs (w/o p2p-circuit)
func (node *Node) Addrs() []multiaddr.Multiaddr {
	addrs := make([]multiaddr.Multiaddr, 0)
	for _, v := range node.Host.Addrs() {
		if v.String() == "/p2p-circuit" {
			continue
		}
		addrs = append(addrs, v)
	}
	return addrs
}

//DHT returns distributed hashed table
func (node *Node) DHT() *dht.IpfsDHT {
	return node.dht
}

// PeersCount return stream count.
// Depreciated
func (node *Node) PeersCount() int32 {
	return int32(len(node.Network().Conns()))
}

// EstablishedPeersCount return handShakeSucceed steam count.
// Depreciated
func (node *Node) EstablishedPeersCount() int32 {
	return node.PeersCount()
}

//Connected returns connected peer ids
func (node *Node) Connected() []peer.ID {
	connections := node.Network().Conns()
	peers := make([]peer.ID, len(connections))
	for i, n := range connections {
		peers[i] = n.RemotePeer()
	}
	return peers
}

// NewNode return new Node according to the config.
func NewNode(ctx context.Context, cfg *medletpb.Config, recvMessageCh chan<- Message) (*Node, error) {
	// check Listen port.
	if err := checkPortAvailable(cfg.Network.Listens); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"listen": cfg.Network.Listens,
		}).Error("Listen port is not available.")
		return nil, err
	}

	networkKey, err := LoadOrNewNetworkKey(cfg)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load network key")
		return nil, err
	}

	lw := int(cfg.Network.ConnMgrLowWaterMark)
	if lw == 0 {
		lw = DefaultConnMgrLowWater
	}
	hw := int(cfg.Network.ConnMgrHighWaterMark)
	if hw == 0 {
		hw = DefaultConnMgrHighWater
	}
	gp := time.Duration(cfg.Network.ConnMgrGracePeriod) * time.Second
	if gp == 0 {
		gp = DefaultConnMgrGracePeriod
	}
	connMgr := connmgr.NewConnManager(lw, hw, gp)

	host, err := libp2p.New(ctx,
		libp2p.Identity(networkKey),
		libp2p.ListenAddrStrings(cfg.Network.Listens...),
		libp2p.ConnectionManager(connMgr),
		//libp2p.BandwidthReporter()
	)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to init libp2p host")
		return nil, err
	}

	logging.Console().WithFields(logrus.Fields{
		"id":    host.ID().Pretty(),
		"addrs": host.Addrs(),
	}).Info("New host started")

	dht, err := dht.New(ctx, host, dhtopts.Protocols(MedDHTProtocolID))
	if err != nil {
		return nil, err
	}

	bCfg, err := NewBootstrapConfig(cfg)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"medletCfg": cfg,
			"err":       err,
		}).Error("failed to make new bootstrap config")
		return nil, err
	}

	cache := path.Join(cfg.Global.Datadir, DefaultCacheFile)
	if cfg.Network.CacheFile != "" {
		cache = cfg.Network.CacheFile
	}

	cachePeriod := DefaultCachePeriod
	if cfg.Network.CachePeriod != 0 {
		cachePeriod = time.Duration(cfg.Network.CachePeriod) * time.Second
	}

	bf := NewBloomFilter(
		bloomFilterOfRecvMessageArgM,
		bloomFilterOfRecvMessageArgK,
		maxCountOfRecvMessageInBloomFiler,
	)

	return &Node{
		Host:            host,
		context:         ctx,
		dht:             dht,
		dhtTicker:       make(chan time.Time),
		messageSender:   newMessageSender(ctx, host, int32(cfg.Network.MaxWriteConcurrency)),
		messageReceiver: newMessageReceiver(ctx, recvMessageCh, bf, int32(cfg.Network.MaxReadConcurrency)),
		chainID:         cfg.Global.ChainId,
		bloomFilter:     bf,
		bootstrapConfig: bCfg,
		cacheFile:       cache,
		cachePeriod:     cachePeriod,
	}, nil
}

// Start host & route table discovery
func (node *Node) Start() error {
	logging.Console().Info("Starting MedService Node...")

	node.SetStreamHandler(MedProtocolID, node.receiveMessage)

	proc, err := node.dht.BootstrapOnSignal(dht.DefaultBootstrapConfig, node.dhtTicker)
	if err != nil {
		return err
	}
	go func() {
		defer proc.Close()
		select {
		case <-node.context.Done():
		case <-node.dht.Context().Done():
		}
	}()

	node.messageSender.Start()
	node.messageReceiver.Start()
	node.Bootstrap()
	node.DHTSync()

	go node.loop()

	logging.Console().WithFields(logrus.Fields{
		"id":                node.ID(),
		"listening address": node.Addrs(),
	}).Info("Started MedService Node.")

	return nil
}

func (node *Node) loop() {
	cacheTicker := time.NewTicker(node.cachePeriod)
	bootstrapTicker := time.NewTicker(node.bootstrapConfig.Period)
	dhtTicker := time.NewTicker(dht.DefaultBootstrapConfig.Period)
	for {
		// route table sync ticker
		select {
		case <-node.context.Done():
			logging.Console().WithFields(logrus.Fields{
				"id":                node.ID(),
				"listening address": node.Addrs(),
			}).Info("Stopping MedService Node...")
			node.Close()
			return
		case <-cacheTicker.C:
			node.SaveCache()
		case <-bootstrapTicker.C:
			node.Bootstrap()
		case <-dhtTicker.C:
			node.DHTSync()
		}
	}
}

//DHTSync run dht bootstrap
func (node *Node) DHTSync() {
	node.dhtTicker <- time.Now()
}

func (node *Node) receiveMessage(s inet.Stream) {
	node.messageReceiver.streamCh <- s
}

func (node *Node) sendMessage(msg *SendMessage) {
	node.messageSender.msgCh <- msg
}

// SendMessageToPeer send message to a peer.
func (node *Node) SendMessageToPeer(msgType string, data []byte, priority int, peerID string) {
	id, err := peer.IDB58Decode(peerID)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":     err,
			"peerID":  peerID,
			"msgType": msgType,
		}).Error("invalid peer id")
		return // ignore
	}

	err = node.Connect(node.context, node.Peerstore().PeerInfo(id))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"id":  id.Pretty(),
			"err": err,
		}).Debug("failed to connect to peer")
		return // TODO
	}

	msg, err := newSendMessage(node.chainID, msgType, data, priority, id)
	if err != nil {
		logging.Console().Debug("failed to make new sendMessage")
		return
	}

	node.sendMessage(msg)
}

//SendMessageToPeers send messages to filtered peers
func (node *Node) SendMessageToPeers(msgType string, data []byte, priority int, filter PeerFilterAlgorithm) (peers []string) {
	msg, err := newSendMessage(node.chainID, msgType, data, priority, "")
	if err != nil {
		logging.Console().Debug("failed to make new sendMessage")
		return
	}

	receivers := filter.Filter(node.Connected())
	for _, p := range receivers {
		msg := msg.copy()
		msg.SetReceiver(p)
		node.sendMessage(msg)
	}

	ids := make([]string, len(receivers))
	for i, p := range receivers {
		ids[i] = p.Pretty()
	}
	return ids
}

// BroadcastMessage broadcast message.
func (node *Node) BroadcastMessage(msgType string, data []byte, priority int) {
	tempMsg, err := newSendMessage(node.chainID, msgType, data, priority, "")
	if err != nil {
		logging.Console().Debug("failed to make new sendMessage")
		return
	}

	receivers := node.Connected() //TODO: minimize broadcast target @drsleepytiger
	for _, p := range receivers {
		msg := tempMsg.copy()
		msg.SetReceiver(p)
		if node.bloomFilter.HasRecvMessage(msg) {
			logging.Console().WithFields(logrus.Fields{
				"hash":    msg.Hash(),
				"peerID":  msg.receiver.Pretty(),
				"msgType": msg.MessageType(),
			}).Debug("avoid duplicate message broadcast")
			continue
		}
		node.sendMessage(msg)
	}
}

//ClosePeer close the connection to peer
func (node *Node) ClosePeer(peerID string, reason error) {
	id, err := peer.IDB58Decode(peerID)
	if err != nil {
		return // ignore
	}
	for _, c := range node.Network().ConnsToPeer(id) {
		c.Close()
	}
}

func (node *Node) addPeer(pi peerstore.PeerInfo, ttl time.Duration) {
	node.Peerstore().AddAddrs(pi.ID, pi.Addrs, ttl)
	go func() {
		ctx, cancel := context.WithTimeout(node.context, 1*time.Second)
		defer cancel()
		pings, err := ping.Ping(ctx, node, pi.ID)
		if err != nil {
			node.Peerstore().ClearAddrs(pi.ID)
			return
		}
		_, ok := <-pings
		if !ok {
			node.Peerstore().ClearAddrs(pi.ID)
			return
		}
		node.dht.Update(node.context, pi.ID)
	}()
}

func (node *Node) addPeersFromProto(peers *netpb.Peers) error {
	for _, p := range peers.Peers {
		pid, err := peer.IDB58Decode(p.Id)
		if err != nil {
			logging.WithFields(logrus.Fields{
				"pid": p.Id,
				"err": err,
			}).Warn("failed to decode pid")
			return err
		}
		addrs := make([]multiaddr.Multiaddr, len(p.Addrs))
		for i, addr := range p.Addrs {
			addrs[i], err = multiaddr.NewMultiaddr(addr)
			if err != nil {
				logging.WithFields(logrus.Fields{
					"add": p.Id,
					"err": err,
				}).Warn("failed to convert multiaddr")
				return err
			}
		}
		node.addPeer(peerstore.PeerInfo{ID: pid, Addrs: addrs}, peerstore.PermanentAddrTTL)
	}
	return nil
}

// loadPeerStoreFromCache load peerstore from cache file
func (node *Node) loadPeerStoreFromCache(cacheFile string) error {
	_, err := os.Stat(cacheFile)
	if err == os.ErrNotExist {
		return nil
	} else if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return err
	}
	str := string(b)

	peers := new(netpb.Peers)
	if err := proto.UnmarshalText(str, peers); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Warn("failed to unmarshal peers from cache file")
		return err
	}

	return node.addPeersFromProto(peers)
}
