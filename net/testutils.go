package net

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-peer"
	"github.com/medibloc/go-medibloc/util/logging"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"net"
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
	StreamReadyCheckInterval         = 100 * time.Millisecond
)

// error for test
var (
	ErrInvalidTestNodeIndex       = errors.New("invalid test node index")
	ErrInvalidTestMedServiceIndex = errors.New("invalid test MedService index")
)

// MedServiceTestManager manages test MedServices
type MedServiceTestManager struct {
	serviceNum      int
	nodeTestManager *NodeTestManager
	medServices     []*MedService
	seedNum         int
}

// NewMedServiceTestManager returns medServicesTestManager
func NewMedServiceTestManager(nodeNum int, seedNum int) *MedServiceTestManager {
	logging.TestHook()
	if nodeNum < 1 {
		return nil
	}
	if nodeNum < seedNum {
		seedNum = nodeNum
	}
	return &MedServiceTestManager{
		nodeTestManager: NewNodeTestManager(nodeNum, seedNum),
		serviceNum:      nodeNum,
		seedNum:         seedNum,
	}
}

// MakeNewTestMedService returns medServices
func (mstm *MedServiceTestManager) MakeNewTestMedService() ([]*MedService, error) {
	nodes, err := mstm.nodeTestManager.MakeNewTestNodes()
	mstm.medServices = make([]*MedService, len(nodes))
	if err != nil {
		return nil, err
	}
	for i, n := range nodes {
		mstm.medServices[i] = &MedService{
			node:       n,
			dispatcher: makeNewTestDispatcher(),
		}
		n.SetMedService(mstm.medServices[i])
	}
	return mstm.medServices, nil
}

// MedService returns MedService of specific index
func (mstm *MedServiceTestManager) MedService(idx int) (*MedService, error) {
	if idx < 0 && idx >= mstm.serviceNum {
		return nil, ErrInvalidTestMedServiceIndex
	}
	return mstm.medServices[idx], nil
}

// StartMedServices starts medServices
func (mstm *MedServiceTestManager) StartMedServices() {
	for _, m := range mstm.medServices {
		m.Start()
	}
}

// StopMedServices stops medServices
func (mstm *MedServiceTestManager) StopMedServices() {
	for _, m := range mstm.medServices {
		m.Stop()
	}
}

// WaitRouteTableSync waits until routing tables are synced
func (mstm *MedServiceTestManager) WaitRouteTableSync() {
	mstm.nodeTestManager.WaitRouteTableSync()
}

// WaitStreamReady waits until streams are ready
func (mstm *MedServiceTestManager) WaitStreamReady() {
	mstm.nodeTestManager.WaitStreamReady()
}

// NodeTestManager manages test nodes
type NodeTestManager struct {
	nodeIDs []peer.ID
	nodeNum int
	nodes   []*Node
	seedNum int
}

// NewNodeTestManager returns nodeTestManager
func NewNodeTestManager(nodeNum int, seedNum int) *NodeTestManager {
	logging.TestHook()
	if nodeNum < 1 {
		return nil
	}
	if nodeNum < seedNum {
		seedNum = nodeNum
	}
	return &NodeTestManager{
		nodeNum: nodeNum,
		seedNum: seedNum,
	}
}

// MakeNewTestNodes returns nodes.
// seed nodes are first seedNum nodes in nodes
func (ntm *NodeTestManager) MakeNewTestNodes() ([]*Node, error) {
	if ntm.nodes != nil {
		return ntm.nodes, nil
	}
	var err error
	ntm.nodes, ntm.nodeIDs, err = makeAndSetNewTestNodes(ntm.nodeNum, ntm.seedNum)
	return ntm.nodes, err
}

// Node returns node of specific index
func (ntm *NodeTestManager) Node(idx int) (*Node, error) {
	if idx < 0 && idx >= ntm.nodeNum {
		return nil, ErrInvalidTestNodeIndex
	}
	return ntm.nodes[idx], nil
}

// StartTestNodes starts nodes
func (ntm *NodeTestManager) StartTestNodes() {
	for _, n := range ntm.nodes {
		n.Start()
	}
}

// StopTestNodes stops nodes
func (ntm *NodeTestManager) StopTestNodes() {
	for _, n := range ntm.nodes {
		n.Stop()
	}
}

// WaitRouteTableSync waits until routing tables are synced
func (ntm *NodeTestManager) WaitRouteTableSync() {
	logging.Console().Debug("WaitRouteTableSync Start...")
	var wg sync.WaitGroup
	for i := range ntm.nodes {
		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			waitRouteTableSyncLoop(n, ntm.nodeIDs)
		}(ntm.nodes[i])
	}
	wg.Wait()
	logging.Console().Debug("WaitRouteTableSync Finish.")
}

// WaitStreamReady waits until streams are ready
func (ntm *NodeTestManager) WaitStreamReady() {
	logging.Console().Debug("WaitStreamReady Start...")
	var wg sync.WaitGroup
	for i := range ntm.nodes {
		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			waitStreamReadyLoop(n, ntm.nodeIDs)
		}(ntm.nodes[i])
	}
	wg.Wait()
	logging.Console().Debug("WaitStreamReady Finish.")
}

func makeNewTestDispatcher() *Dispatcher {
	logging.TestHook()
	return NewDispatcher()
}

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

	randomListen := makeRandomListen()
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

func makeRandomListen() []string {
	lis, _ := net.Listen("tcp", ":0")
	addr := lis.Addr().String()
	lis.Close()
	return []string{addr}
}

func waitRouteTableSyncLoop(node *Node, nodeIDs []peer.ID) {
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
		// fail if (remainIteration * RouteTableCheckInterval) Duration left
		if remainIteration < 1 && len(ids) > 0 {
			logging.Console().WithFields(logrus.Fields{
				"node ID":                          node.ID(),
				"routeTable not synced node count": len(ids),
			}).Warn("route table not synced in time")
			return
		}
	}
}

func waitStreamReadyLoop(node *Node, nodeIDs []peer.ID) {
	ids := make([]peer.ID, len(nodeIDs))
	copy(ids, nodeIDs)
	remainIteration := 100

	for len(ids) > 0 {
		i := 0
		for _, v := range ids {
			if node.streamManager.Find(v) == nil && node.id != v {
				ids[i] = v
				i++
			}
		}
		ids = ids[:i]

		time.Sleep(StreamReadyCheckInterval)

		remainIteration--
		// fail if (remainIteration * RouteTableCheckInterval) Duration left
		if remainIteration < 1 && len(ids) > 0 {
			logging.Console().WithFields(logrus.Fields{
				"node ID":               node.ID(),
				"streams are not ready": len(ids),
			}).Warn("streams not ready in time")
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
