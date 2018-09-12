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

package medlet

import (
	goNet "net"

	"net/http"
	_ "net/http/pprof" // add pprof
	"time"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/rpc"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/sync"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

var (
	metricsMedstartGauge = metrics.GetOrRegisterGauge("med.start", nil)
)

//DefaultTxMap is default map of transactions.
var DefaultTxMap = core.TxFactory{
	core.TxOpTransfer:            core.NewTransferTx,
	core.TxOpAddRecord:           core.NewAddRecordTx,
	core.TxOpVest:                core.NewVestTx,
	core.TxOpWithdrawVesting:     core.NewWithdrawVestingTx,
	core.TxOpAddCertification:    core.NewAddCertificationTx,
	core.TxOpRevokeCertification: core.NewRevokeCertificationTx,

	dpos.TxOpBecomeCandidate: dpos.NewBecomeCandidateTx,
	dpos.TxOpQuitCandidacy:   dpos.NewQuitCandidateTx,
	dpos.TxOpVote:            dpos.NewVoteTx,
}

// Medlet manages blockchain services.
type Medlet struct {
	config             *medletpb.Config
	genesis            *corepb.Genesis
	netService         net.Service
	rpc                *rpc.Server
	storage            storage.Storage
	blockManager       *core.BlockManager
	transactionManager *core.TransactionManager
	consensus          *dpos.Dpos
	eventEmitter       *core.EventEmitter
	syncService        *sync.Service
}

// New returns a new medlet.
func New(cfg *medletpb.Config) (*Medlet, error) {
	genesis, err := core.LoadGenesisConf(cfg.Chain.Genesis)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"path": cfg.Chain.Genesis,
			"err":  err,
		}).Error("Failed to load genesis config.")
		return nil, err
	}

	ns, err := net.NewMedService(cfg)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create net service.")
		return nil, err
	}

	rpc := rpc.New(cfg)

	//stor, err := storage.NewLeveldbStorage(cfg.Global.Datadir)
	stor, err := storage.NewRocksStorage(cfg.Global.Datadir)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create leveldb storage.")
		return nil, err
	}

	bm, err := core.NewBlockManager(cfg)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create BlockManager.")
		return nil, err
	}

	tm := core.NewTransactionManager(cfg)

	consensus := dpos.New(int(genesis.Meta.DynastySize))

	ss := sync.NewService(cfg.Sync)

	return &Medlet{
		config:             cfg,
		genesis:            genesis,
		netService:         ns,
		rpc:                rpc,
		storage:            stor,
		blockManager:       bm,
		transactionManager: tm,
		consensus:          consensus,
		eventEmitter:       core.NewEventEmitter(40960),
		syncService:        ss,
	}, nil
}

// Setup sets up medlet.
func (m *Medlet) Setup() error {
	logging.Console().Info("Setting up Medlet...")

	m.rpc.Setup(m.blockManager, m.transactionManager, m.eventEmitter)

	err := m.blockManager.Setup(m.genesis, m.storage, m.netService, m.consensus)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to setup BlockManager.")
		return err
	}

	m.transactionManager.Setup(m.netService)

	err = m.consensus.Setup(m.config, m.genesis, m.blockManager, m.transactionManager)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to setup consensus.")
		return err
	}

	m.transactionManager.InjectEmitter(m.eventEmitter)
	m.blockManager.InjectEmitter(m.eventEmitter)
	m.blockManager.InjectTransactionManager(m.transactionManager)

	m.blockManager.SetTxMap(DefaultTxMap)

	m.blockManager.InjectSyncService(m.syncService)
	m.syncService.Setup(m.netService, m.blockManager)

	logging.Console().Info("Set up Medlet.")
	return nil
}

// StartPprof start pprof http listen
func (m *Medlet) StartPprof(listen string) error {
	if len(listen) > 0 {
		conn, err := goNet.DialTimeout("tcp", listen, time.Second*1)
		if err == nil {
			logging.Console().WithFields(logrus.Fields{
				"listen": listen,
				"err":    err,
			}).Error("Failed to start pprof")
			conn.Close()
			return err
		}

		go func() {
			logging.Console().WithFields(logrus.Fields{
				"listen": listen,
			}).Info("Starting pprof...")
			http.ListenAndServe(listen, nil)
		}()
	}
	return nil
}

// Start starts the services of the medlet.
func (m *Medlet) Start() error {
	err := m.netService.Start()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to start net service.")
		return err
	}

	err = m.rpc.Start()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to start rpc service.")
		return err
	}

	m.eventEmitter.Start()

	m.blockManager.Start()

	m.transactionManager.Start()

	m.consensus.Start()

	m.syncService.Start()

	metricsMedstartGauge.Update(1)

	logging.Console().Info("Started Medlet.")
	return nil
}

// Stop stops the services of the medlet.
func (m *Medlet) Stop() {
	err := m.storage.Close()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to close storage")
	}

	m.eventEmitter.Stop()

	m.netService.Stop()

	m.blockManager.Stop()

	m.transactionManager.Stop()

	m.rpc.Stop()

	m.consensus.Stop()

	m.syncService.Stop()

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
func (m *Medlet) RPC() *rpc.Server {
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

// Consensus returns consensus
func (m *Medlet) Consensus() core.Consensus {
	return m.consensus
}

// EventEmitter returns event emitter.
func (m *Medlet) EventEmitter() *core.EventEmitter {
	return m.eventEmitter
}

//SyncService returns sync service
func (m *Medlet) SyncService() *sync.Service {
	return m.syncService
}
