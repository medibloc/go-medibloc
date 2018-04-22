package sync

import (
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
)

type Service struct {
	Seeding  *Seeding
	Download *Download

	quitCh chan bool
}

func NewService(m medlet.Medlet) *Service {
	return &Service{
		Seeding:  newSeeding(m.NetService(), m.blockManager),
		Download: newDownload(m.NetService(), m.blockManager),
		quitCh:   nil,
	}
}

func (ss *Service) Start() {
	ss.Seeding.Start()
	go ss.startLoop()
}

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
			ss.Seeding.Stop()
			ss.Download.Stop()
			logging.Console().Info("Stopped Sync Service.")
			return
		}
	}
}

func (ss *Service) activeDownload(chunkSize uint64) {
	if ss.Download.activated == true {
		logging.Console().Error("Sync: Download Manager is already activated.")
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
