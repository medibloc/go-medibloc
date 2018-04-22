package sync

import (
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
)

//Service Sync
type Service struct {
	Seeding  *seeding
	Download *download

	quitCh chan bool
}

//NewService Generate syncService
func NewService(m medlet.Medlet) *Service {
	return &Service{
		Seeding:  newSeeding(m.NetService(), m.blockManager),
		Download: newDownload(m.NetService(), m.blockManager),
		quitCh:   nil,
	}
}

// Start Sync Service
func (ss *Service) Start() {
	ss.Seeding.start()
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

func (ss *Service) activeDownload(chunkSize uint64) {
	if ss.Download.activated == true {
		logging.Console().Error("Sync: download Manager is already activated.")
		return
	}
	ss.Download.setup(chunkSize)
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
