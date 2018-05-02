package sync

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

//Service Sync
type Service struct {
	config *medletpb.SyncConfig

	Seeding  *seeding
	Download *download

	quitCh chan bool
}

//NewService returns new syncService
func NewService(config *medletpb.SyncConfig) *Service {
	return &Service{
		config:   config,
		Seeding:  nil,
		Download: nil,
		quitCh:   make(chan bool, 1),
	}
}

//Setup makes seeding and block manager on syncService
func (ss *Service) Setup(netService net.Service, bm BlockManager) {
	ss.Seeding = newSeeding(ss.config)
	ss.Seeding.setup(netService, bm)
	ss.Download = newDownload(ss.config)
	ss.Download.setup(netService, bm)
}

// Start Sync Service
func (ss *Service) Start() {
	logging.Console().Info("SyncService is started")
	ss.Seeding.start()
	logging.Console().Debug("Seeding start OK")
	go ss.startLoop()
}

//Stop Sync Service
func (ss *Service) Stop() {
	ss.quitCh <- true
}

func (ss *Service) startLoop() {

	logging.Console().Info("Sync service is started.")
	//timerChan := time.NewTicker(time.Second).C
	for {
		select {
		//case <-timerChan:
		//	metricsCachedSync.Update(int64(len(ss.messageCh)))
		case <-ss.quitCh:
			ss.Seeding.stop()
			ss.Download.stop()
			logging.Console().Info("Stopped Sync Service.")
			return
		}
	}
}

//ActiveDownload start download manager
func (ss *Service) ActiveDownload() {
	if ss.Download.activated == true {
		logging.Console().Error("Sync: download Manager is already activated.")
		return
	}
	logging.Info("Sync: Download manager started.")
	ss.Download.start()
}

func generateHashTrie(hashes []common.Hash) *trie.Trie {
	store, err := storage.NewMemoryStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to create memory storage")
		return nil
	}

	hashTrie, err := trie.NewTrie(nil, store)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to create merkle tree")
		return nil
	}

	for _, h := range hashes {
		hashTrie.Put(h.Bytes(), h.Bytes())
	}

	return hashTrie

}
