package medlet

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet/pb"
	mednet "github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/rpc"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	m "github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

var (
	metricsMedstartGauge   = m.GetOrRegisterGauge("med.start", nil)
	transactionManagerSize = 1280
)

// Medlet manages blockchain services.
type Medlet struct {
	bs         *core.BlockSubscriber
	config     *medletpb.Config
	miner      *core.Miner
	netService mednet.Service
	rpc        rpc.GRPCServer
	txMgr      *core.TransactionManager
}

type rpcBridge struct {
	bm *core.BlockManager
}

// BlockManager return core.BlockManager
func (gb *rpcBridge) BlockManager() *core.BlockManager {
	return gb.bm
}

// New returns a new medlet.
func New(config *medletpb.Config) (*Medlet, error) {
	return &Medlet{
		config: config,
	}, nil
}

// Setup sets up medlet.
func (m *Medlet) Setup() {
	var err error
	logging.Console().Info("Setting up Medlet...")

	m.netService, err = mednet.NewMedService(m)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to setup net service.")
	}

	logging.Console().Info("Set up Medlet.")
}

// Start starts the services of the medlet.
func (m *Medlet) Start() {
	if err := m.netService.Start(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to start net service.")
		return
	}

	metricsMedstartGauge.Update(1)

	s, err := storage.NewLeveldbStorage(m.config.Chain.Datadir)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to create leveldb storage")
		return
	}

	m.txMgr = core.NewTransactionManager(m, transactionManagerSize)
	m.txMgr.RegisterInNetwork(m.netService)
	m.txMgr.Start()

	bp, bc, err := core.GetBlockPoolBlockChain(s)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to create block pool or block chain")
		return
	}
	m.bs = core.StartBlockSubscriber(m.netService, bp, bc)

	if m.Config().Chain.StartMine {
		m.miner = core.StartMiner(m.netService, bc, m.txMgr)
	}

	m.rpc = rpc.NewServer(&rpcBridge{bm: m.bs.BlockManager()})
	m.rpc.Start(m.config.Rpc.RpcListen[0]) // TODO

	logging.Console().Info("Started Medlet.")
}

// Stop stops the services of the medlet.
func (m *Medlet) Stop() {
	if m.netService != nil {
		m.netService.Stop()
		m.netService = nil
	}

	m.txMgr.Stop()

	m.bs.StopBlockSubscriber()
	if m.miner != nil {
		m.miner.StopMiner()
	}

	m.rpc.Stop()

	logging.Console().Info("Stopped Medlet.")
}

// Config returns medlet configuration.
func (m *Medlet) Config() *medletpb.Config {
	return m.config
}
