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
	"context"
	"sync"
	"time"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/logging"
)

//Service is the service for sync service
type Service struct {
	ctx context.Context

	netService net.Service
	bm         BlockManager
	messageCh  chan net.Message

	mu          sync.Mutex
	downloading bool

	downloadCtx      context.Context
	downloadCancel   context.CancelFunc

	targetHeight     uint64
	targetHash       []byte
	baseBlock        *core.BlockData
	subscribeMap     *sync.Map // key: queryID, value: blockHeight
	numberOfRequests int
	downloadErrCh    chan error

	responseTimeLimit   time.Duration
	numberOfRetries     int
	activeDownloadLimit int
}

//IsDownloadActivated return status of activation
func (s *Service) IsDownloadActivated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.downloading
}

// NewService returns new syncService
func NewService(cfg *medletpb.SyncConfig) *Service {
	responseTimeLimit := time.Duration(cfg.ResponseTimeLimit) * time.Second
	if responseTimeLimit == 0 {
		responseTimeLimit = DefaultResponseTimeLimit
	}
	numberOfRetries := int(cfg.NumberOfRetries)
	if numberOfRetries == 0 {
		numberOfRetries = DefaultNumberOfRetry
	}
	activeDownloadLimit := int(cfg.ActiveDownloadLimit)
	if activeDownloadLimit == 0 {
		activeDownloadLimit = DefaultActiveDownloadLimit
	}

	return &Service{
		netService:       nil,
		bm:               nil,
		messageCh:        make(chan net.Message, 128),
		mu:               sync.Mutex{},
		downloading:      false,
		targetHeight:     0,
		numberOfRequests: 0,

		responseTimeLimit:   responseTimeLimit,
		numberOfRetries:     numberOfRetries,
		activeDownloadLimit: activeDownloadLimit,
	}
}

// Setup makes seeding and block manager on syncService
func (s *Service) Setup(netService net.Service, bm BlockManager) {
	s.netService = netService
	s.bm = bm
}

// Start Sync Service
func (s *Service) Start(ctx context.Context) {
	s.ctx = ctx
	logging.Console().Info("SyncService is started.")

	s.netService.Register(net.NewSubscriber(s, s.messageCh, false, BaseSearch, net.MessageWeightZero))
	s.netService.Register(net.NewSubscriber(s, s.messageCh, false, BlockRequest, net.MessageWeightZero))

	go s.loop()
}

func (s *Service) stop() {
	s.netService.Deregister(net.NewSubscriber(s, s.messageCh, false, BaseSearch, net.MessageWeightZero))
	s.netService.Deregister(net.NewSubscriber(s, s.messageCh, false, BlockRequest, net.MessageWeightZero))

	if s.subscribeMap != nil {
		s.subscribeMap.Range(func(key, _ interface{}) bool {
			qID := key.(string)
			s.netService.Deregister(net.NewSubscriber(s, s.messageCh, false, qID, net.MessageWeightZero))
			return true
		})
	}
	logging.Console().Info("SyncService is started.")
}

func (s *Service) loop() {
	for {
		select {
		case <-s.ctx.Done():
			s.stop()
			return
		case msg := <-s.messageCh:
			switch msg.MessageType() {
			case BaseSearch:
				go s.handleFindBaseRequest(msg)
			case BlockRequest:
				go s.handleBlockByHeightRequest(msg)
			}
		}
	}
}
