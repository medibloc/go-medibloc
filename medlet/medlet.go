package medlet

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet/pb"
	mednet "github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	m "github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

var (
	metricsMedstartGauge = m.GetOrRegisterGauge("med.start", nil)
)

// Medlet manages blockchain services.
type Medlet struct {
	config     *medletpb.Config
	netService mednet.Service
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
	logging.Console().Info("Setuping Medlet...")

	m.netService, err = mednet.NewMedService(m)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to setup net service.")
	}

	logging.Console().Info("Setuped Medlet.")
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

	bp, bc, err := core.GetBlockPoolBlockChain(s)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to create block pool or block chain")
		return
	}
	err = core.StartBlockSubscriber(m.netService, bp, bc)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to start block subscriber")
		return
	}

	logging.Console().Info("Started Medlet.")
}

// Stop stops the services of the medlet.
func (m *Medlet) Stop() {
	if m.netService != nil {
		m.netService.Stop()
		m.netService = nil
	}

	core.StopBlockSubscriber()

	logging.Console().Info("Stopped Medlet.")
}

// Config returns medlet configuration.
func (m *Medlet) Config() *medletpb.Config {
	return m.config
}
