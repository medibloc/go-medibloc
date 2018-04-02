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
func NewMedService(m Medlet) (*MedService, error) {
	if networkConf := m.Config().GetNetwork(); networkConf == nil {
		logging.Console().Fatal("config.conf should has network")
		return nil, ErrConfigLackNetWork
	}
	node, err := NewNode(NewP2PConfig(m))
	if err != nil {
		return nil, err
	}

	ms := &MedService{
		node:       node,
		dispatcher: NewDispatcher(),
	}
	node.SetMedService(ms)

	return ms, nil
}

// Node return the peer node
func (ms *MedService) Node() *Node {
	return ms.node
}

// Start start p2p manager.
func (ms *MedService) Start() error {
	logging.Console().Info("Starting MedService...")

	// start dispatcher.
	ms.dispatcher.Start()

	// start node.
	if err := ms.node.Start(); err != nil {
		ms.dispatcher.Stop()
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to start MedService.")
		return err
	}

	logging.Console().Info("Started MedService.")
	return nil
}

// Stop stop p2p manager.
func (ms *MedService) Stop() {
	logging.Console().Info("Stopping MedService...")

	ms.node.Stop()
	ms.dispatcher.Stop()
}

// Register register the subscribers.
func (ms *MedService) Register(subscribers ...*Subscriber) {
	ms.dispatcher.Register(subscribers...)
}

// Deregister Deregister the subscribers.
func (ms *MedService) Deregister(subscribers ...*Subscriber) {
	ms.dispatcher.Deregister(subscribers...)
}

// PutMessage put message to dispatcher.
func (ms *MedService) PutMessage(msg Message) {
	ms.dispatcher.PutMessage(msg)
}

// Broadcast message.
func (ms *MedService) Broadcast(name string, msg Serializable, priority int) {
	ms.node.BroadcastMessage(name, msg, priority)
}

// Relay message.
func (ms *MedService) Relay(name string, msg Serializable, priority int) {
	ms.node.RelayMessage(name, msg, priority)
}

// BroadcastNetworkID broadcast networkID when changed.
func (ms *MedService) BroadcastNetworkID(msg []byte) {
	// TODO: networkID.
}

// BuildRawMessageData return the raw MedMessage content data.
func (ms *MedService) BuildRawMessageData(data []byte, msgName string) []byte {
	message, err := NewMedMessage(ms.node.config.ChainID, DefaultReserved, 0, msgName, data)
	if err != nil {
		return nil
	}

	return message.Content()
}

// SendMsg send message to a peer.
func (ms *MedService) SendMsg(msgName string, msg []byte, target string, priority int) error {
	return ms.node.SendMessageToPeer(msgName, msg, priority, target)
}

// SendMessageToPeers send message to peers.
func (ms *MedService) SendMessageToPeers(messageName string, data []byte, priority int, filter PeerFilterAlgorithm) []string {
	return ms.node.streamManager.SendMessageToPeers(messageName, data, priority, filter)
}

// SendMessageToPeer send message to a peer.
func (ms *MedService) SendMessageToPeer(messageName string, data []byte, priority int, peerID string) error {
	return ms.node.SendMessageToPeer(messageName, data, priority, peerID)
}

// ClosePeer close the stream to a peer.
func (ms *MedService) ClosePeer(peerID string, reason error) {
	ms.node.streamManager.CloseStream(peerID, reason)
}
