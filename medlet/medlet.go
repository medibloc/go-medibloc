package medlet

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/rpc"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

var (
	metricsMedstartGauge = metrics.GetOrRegisterGauge("med.start", nil)
)

// Medlet manages blockchain services.
type Medlet struct {
	config             *medletpb.Config
	genesis            *corepb.Genesis
	netService         net.Service
	rpc                rpc.GRPCServer
	storage            storage.Storage
	blockManager       *core.BlockManager
	transactionManager *core.TransactionManager
}

// New returns a new medlet.
func New(cfg *medletpb.Config) (*Medlet, error) {
	genesis, err := core.LoadGenesisConf(cfg.Chain.Genesis)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"path": cfg.Chain.Genesis,
			"err":  err,
		}).Fatal("Failed to load genesis config.")
		return nil, err
	}

	ns, err := net.NewMedService(cfg)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to create net service.")
		return nil, err
	}

	// TODO @cl9200 rpc new

	stor, err := storage.NewLeveldbStorage(cfg.Chain.Datadir)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to create leveldb storage.")
		return nil, err
	}

	bm, err := core.NewBlockManager(cfg)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to create BlockManager.")
		return nil, err
	}

	tm := core.NewTransactionManager(cfg)

	return &Medlet{
		config:             cfg,
		genesis:            genesis,
		netService:         ns,
		storage:            stor,
		blockManager:       bm,
		transactionManager: tm,
	}, nil
}

// Setup sets up medlet.
func (m *Medlet) Setup() error {
	logging.Console().Info("Setting up Medlet...")

	err := m.blockManager.Setup(m.genesis, m.storage, m.netService)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to setup BlockManager.")
		return err
	}

	m.transactionManager.Setup(m.netService)

	logging.Console().Info("Set up Medlet.")
	return nil
}

// Start starts the services of the medlet.
func (m *Medlet) Start() error {
	err := m.netService.Start()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to start net service.")
		return err
	}

	m.blockManager.Start()

	m.transactionManager.Start()

	// TODO @cl9200 rpc start
	//m.rpc = rpc.NewServer(&rpcBridge{bm: m.bs.BlockManager(), txMgr: m.txMgr}, m.config.Rpc.RpcListen[0]) // TODO choose index
	//m.rpc.Start()
	//m.rpc.RunGateway(m.config.Rpc.HttpListen[0]) // TODO choose index

	metricsMedstartGauge.Update(1)

	logging.Console().Info("Started Medlet.")
	return nil
}

// Stop stops the services of the medlet.
func (m *Medlet) Stop() {
	m.netService.Stop()

	m.blockManager.Stop()

	m.transactionManager.Stop()

	// TODO @cl9200 rpc stop
	m.rpc.Stop()

	logging.Console().Info("Stopped Medlet.")
}

// Config returns medlet configuration.
func (m *Medlet) Config() *medletpb.Config {
	return m.config
}

// Genesis returns genesis config.
func (m *Medlet) Genesis() *corepb.Genesis {
	return m.genesis
}

// NetService returns NetService.
func (m *Medlet) NetService() net.Service {
	return m.netService
}

// RPC returns RPC.
func (m *Medlet) RPC() rpc.GRPCServer {
	return m.rpc
}

// Storage returns storage.
func (m *Medlet) Storage() storage.Storage {
	return m.storage
}

// BlockManager returns BlockManager.
func (m *Medlet) BlockManager() *core.BlockManager {
	return m.blockManager
}

// TransactionManager returns TransactionManager.
func (m *Medlet) TransactionManager() *core.TransactionManager {
	return m.transactionManager
}
