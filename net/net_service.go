package net

import (
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// MedService service for medibloc p2p network
type MedService struct {
	node       *Node
	dispatcher *Dispatcher
}

// NewMedService create netService
func NewMedService(n Medlet) (*MedService, error) {
	if networkConf := n.Config().GetNetwork(); networkConf == nil {
		logging.Console().Fatal("config.conf should has network")
		return nil, ErrConfigLackNetWork
	}
	node, err := NewNode(NewP2PConfig(n))
	if err != nil {
		return nil, err
	}

	ns := &MedService{
		node:       node,
		dispatcher: NewDispatcher(),
	}
	node.SetMedService(ns)

	return ns, nil
}

// Node return the peer node
func (ns *MedService) Node() *Node {
	return ns.node
}

// Start start p2p manager.
func (ns *MedService) Start() error {
	logging.Console().Info("Starting MedService...")

	// start dispatcher.
	ns.dispatcher.Start()

	// start node.
	if err := ns.node.Start(); err != nil {
		ns.dispatcher.Stop()
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to start MedService.")
		return err
	}

	logging.Console().Info("Started MedService.")
	return nil
}

// Stop stop p2p manager.
func (ns *MedService) Stop() {
	logging.Console().Info("Stopping MedService...")

	ns.node.Stop()
	ns.dispatcher.Stop()
}

// Register register the subscribers.
func (ns *MedService) Register(subscribers ...*Subscriber) {
	ns.dispatcher.Register(subscribers...)
}

// Deregister Deregister the subscribers.
func (ns *MedService) Deregister(subscribers ...*Subscriber) {
	ns.dispatcher.Deregister(subscribers...)
}

// PutMessage put message to dispatcher.
func (ns *MedService) PutMessage(msg Message) {
	ns.dispatcher.PutMessage(msg)
}

// Broadcast message.
func (ns *MedService) Broadcast(name string, msg Serializable, priority int) {
	ns.node.BroadcastMessage(name, msg, priority)
}

// Relay message.
func (ns *MedService) Relay(name string, msg Serializable, priority int) {
	ns.node.RelayMessage(name, msg, priority)
}

// BroadcastNetworkID broadcast networkID when changed.
func (ns *MedService) BroadcastNetworkID(msg []byte) {
	// TODO: networkID.
}

// BuildRawMessageData return the raw MedMessage content data.
func (ns *MedService) BuildRawMessageData(data []byte, msgName string) []byte {
	message, err := NewMedMessage(ns.node.config.ChainID, DefaultReserved, 0, msgName, data)
	if err != nil {
		return nil
	}

	return message.Content()
}

// SendMsg send message to a peer.
func (ns *MedService) SendMsg(msgName string, msg []byte, target string, priority int) error {
	return ns.node.SendMessageToPeer(msgName, msg, priority, target)
}

// SendMessageToPeers send message to peers.
func (ns *MedService) SendMessageToPeers(messageName string, data []byte, priority int, filter PeerFilterAlgorithm) []string {
	return ns.node.streamManager.SendMessageToPeers(messageName, data, priority, filter)
}

// SendMessageToPeer send message to a peer.
func (ns *MedService) SendMessageToPeer(messageName string, data []byte, priority int, peerID string) error {
	return ns.node.SendMessageToPeer(messageName, data, priority, peerID)
}

// ClosePeer close the stream to a peer.
func (ns *MedService) ClosePeer(peerID string, reason error) {
	ns.node.streamManager.CloseStream(peerID, reason)
}
