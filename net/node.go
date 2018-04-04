package net

import (
	"context"

	"errors"
	"fmt"
	"net"

	crypto "github.com/libp2p/go-libp2p-crypto"
	libnet "github.com/libp2p/go-libp2p-net"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/medibloc/go-medibloc/util/logging"
	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

const letterBytes = "0123456789ABCDEF0123456789ABCDE10123456789ABCDEF0123456789ABCDEF"

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// Error types
var (
	ErrPeerIsNotConnected = errors.New("peer is not connected")
)

// Node the node can be used as both the client and the server
type Node struct {
	synchronizing bool
	quitCh        chan bool
	netService    *MedService
	config        *Config
	context       context.Context
	id            peer.ID
	networkKey    crypto.PrivKey
	network       *swarm.Network
	host          *basichost.BasicHost
	streamManager *StreamManager
	routeTable    *RouteTable
}

// NewNode return new Node according to the config.
func NewNode(config *Config) (*Node, error) {
	// check Listen port.
	if err := checkPortAvailable(config.Listen); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"listen": config.Listen,
		}).Error("Listen port is not available.")
		return nil, err
	}

	node := &Node{
		quitCh:        make(chan bool, 10),
		config:        config,
		context:       context.Background(),
		streamManager: NewStreamManager(),
		synchronizing: false,
	}

	initP2PNetworkKey(config, node)
	initP2PRouteTable(config, node)

	if err := initP2PSwarmNetwork(config, node); err != nil {
		return nil, err
	}

	return node, nil
}

// Start host & route table discovery
func (node *Node) Start() error {
	logging.Console().Info("Starting MedService Node...")

	node.streamManager.Start()

	if err := node.startHost(); err != nil {
		return err
	}

	node.routeTable.Start()

	logging.Console().WithFields(logrus.Fields{
		"id":                node.ID(),
		"listening address": node.host.Addrs(),
	}).Info("Started MedService Node.")

	return nil
}

// Stop stop a node.
func (node *Node) Stop() {
	logging.Console().WithFields(logrus.Fields{
		"id":                node.ID(),
		"listening address": node.host.Addrs(),
	}).Info("Stopping MedService Node...")

	node.routeTable.Stop()
	node.stopHost()
	node.streamManager.Stop()
}

func (node *Node) startHost() error {
	// add nat manager
	options := &basichost.HostOpts{}
	options.NATManager = basichost.NewNATManager(node.network)
	host, err := basichost.NewHost(node.context, node.network, options)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":            err,
			"listen address": node.config.Listen,
		}).Error("Failed to start node.")
		return err
	}

	host.SetStreamHandler(MedProtocolID, node.onStreamConnected)
	node.host = host

	return nil
}

func (node *Node) stopHost() {
	node.network.Close()

	if node.host == nil {
		return
	}

	node.host.Close()
}

// Config return node config.
func (node *Node) Config() *Config {
	return node.config
}

// SetMedService set netService
func (node *Node) SetMedService(ns *MedService) {
	node.netService = ns
}

// ID return node ID.
func (node *Node) ID() string {
	return node.id.Pretty()
}

// IsSynchronizing return node synchronizing
func (node *Node) IsSynchronizing() bool {
	return node.synchronizing
}

// SetSynchronizing set node synchronizing.
func (node *Node) SetSynchronizing(synchronizing bool) {
	node.synchronizing = synchronizing
}

// PeersCount return stream count.
func (node *Node) PeersCount() int32 {
	return node.streamManager.Count()
}

// RouteTable return route table.
func (node *Node) RouteTable() *RouteTable {
	return node.routeTable
}

func initP2PNetworkKey(config *Config, node *Node) {
	// init p2p network key.
	networkKey, err := LoadNetworkKeyFromFileOrCreateNew(config.PrivateKeyPath)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":        err,
			"NetworkKey": config.PrivateKeyPath,
		}).Warn("Failed to load network private key from file.")
	}

	node.networkKey = networkKey
	node.id, err = peer.IDFromPublicKey(networkKey.GetPublic())
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":        err,
			"NetworkKey": config.PrivateKeyPath,
		}).Warn("Failed to generate ID from network key file.")
	}
}

func initP2PRouteTable(config *Config, node *Node) error {
	// init p2p route table.
	node.routeTable = NewRouteTable(config, node)
	return nil
}

func initP2PSwarmNetwork(config *Config, node *Node) error {
	// init p2p multiaddr and swarm network.
	multiaddrs := make([]multiaddr.Multiaddr, len(config.Listen))
	for idx, v := range node.config.Listen {
		tcpAddr, err := net.ResolveTCPAddr("tcp", v)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":            err,
				"listen address": v,
			}).Error("Invalid listen address.")
			return err
		}

		addr, err := multiaddr.NewMultiaddr(
			fmt.Sprintf(
				"/ip4/%s/tcp/%d",
				tcpAddr.IP,
				tcpAddr.Port,
			),
		)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":            err,
				"listen address": v,
			}).Error("Invalid listen address.")
			return err
		}

		multiaddrs[idx] = addr
	}

	network, err := swarm.NewNetwork(
		node.context,
		multiaddrs,
		node.id,
		node.routeTable.peerStore,
		nil, // TODO: integrate metrics.Reporter.
	)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":            err,
			"listen address": config.Listen,
			"node.id":        node.id.Pretty(),
		}).Error("Failed to create swarm network.")
		return err
	}
	node.network = network
	return nil
}

func (node *Node) onStreamConnected(s libnet.Stream) {
	node.streamManager.Add(s, node)
}

// SendMessageToPeer send message to a peer.
func (node *Node) SendMessageToPeer(messageName string, data []byte, priority int, peerID string) error {
	stream := node.streamManager.FindByPeerID(peerID)
	if stream == nil {
		logging.WithFields(logrus.Fields{
			"pid": peerID,
			"err": ErrPeerIsNotConnected,
		}).Debug("Failed to send msg")
		return ErrPeerIsNotConnected
	}

	return stream.SendMessage(messageName, data, priority)
}

// BroadcastMessage broadcast message.
func (node *Node) BroadcastMessage(messageName string, data Serializable, priority int) {
	// node can not broadcast or relay message if it is in synchronizing.
	if node.synchronizing {
		return
	}

	node.streamManager.BroadcastMessage(messageName, data, priority)
}

// RelayMessage relay message.
func (node *Node) RelayMessage(messageName string, data Serializable, priority int) {
	// node can not broadcast or relay message if it is in synchronizing.
	if node.synchronizing {
		return
	}

	node.streamManager.RelayMessage(messageName, data, priority)
}
