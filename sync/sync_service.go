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

package sync

import (
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
	logging.Console().Info("SyncService is started.")
	ss.Seeding.start()
}

//Stop Sync Service
func (ss *Service) Stop() {
	ss.Seeding.stop()
	ss.Download.stop()
	logging.Console().Info("SyncService is stopped.")
}

//ActiveDownload start download manager
func (ss *Service) ActiveDownload() error {
	if ss.Download.IsActivated() == true {
		return ErrAlreadyDownlaodActivated
	}
	ss.Download.start()
	logging.Info("Sync: Download manager started.")

	return nil
}

//IsActiveDownload return download isActivated
func (ss *Service) IsActiveDownload() bool {
	return ss.Download.IsActivated()
}

func generateHashTrie(hashes [][]byte) *trie.Trie {
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
		hashTrie.Put(h, h)
	}

	return hashTrie

}
