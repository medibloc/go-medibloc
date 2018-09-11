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
	"errors"

	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

const maxNumberOfBlocks = 3600 // number of blocks for 3 hours

type seeding struct {
	netService net.Service
	bm         BlockManager
	quitCh     chan bool

	mu        sync.Mutex
	activated bool

	messageCh    chan net.Message
	minChunkSize uint64
	maxChunkSize uint64
	semaphore    chan bool
}

func newSeeding(config *medletpb.SyncConfig) *seeding {
	return &seeding{
		netService:   nil,
		bm:           nil,
		quitCh:       make(chan bool, 2),
		activated:    false,
		messageCh:    make(chan net.Message, 128),
		minChunkSize: config.SeedingMinChunkSize,
		maxChunkSize: config.SeedingMaxChunkSize,
		semaphore:    make(chan bool, config.SeedingMaxConcurrentPeers),
	}
}

//IsActivated return status of activation
func (s *seeding) IsActivated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activated
}

func (s *seeding) setup(netService net.Service, bm BlockManager) {
	s.netService = netService
	s.bm = bm
}

func (s *seeding) start() {
	logging.Console().Info("Sync: Seeding manager is started.")
	s.netService.Register(net.NewSubscriber(s, s.messageCh, false, net.SyncMetaRequest, net.MessageWeightZero))
	s.netService.Register(net.NewSubscriber(s, s.messageCh, false, net.SyncBlockChunkRequest, net.MessageWeightZero))

	s.mu.Lock()
	s.activated = true
	s.mu.Unlock()

	go s.startLoop()
}

func (s *seeding) stop() {
	s.netService.Deregister(net.NewSubscriber(s, s.messageCh, false, net.SyncMetaRequest, net.MessageWeightZero))
	s.netService.Deregister(net.NewSubscriber(s, s.messageCh, false, net.SyncBlockChunkRequest, net.MessageWeightZero))
	s.quitCh <- true
}

func (s *seeding) startLoop() {
	for {
		select {
		case <-s.quitCh:
			logging.Console().Info("Sync: Seeding manager is stopped.")
			s.mu.Lock()
			s.activated = false
			s.mu.Unlock()
			return
		case message := <-s.messageCh:
			switch message.MessageType() {
			case net.SyncMetaRequest:
				s.sendRootHashMeta(message)
			case net.SyncBlockChunkRequest:
				select {
				case s.semaphore <- true:
					go func() {
						s.sendBlockChunk(message)
						<-s.semaphore
					}()
				default:
				}
				//				if s.nConcurrentPeers < s.maxConcurrentPeers {
				//				go s.sendBlockChunk(message)
				//		}
			}
		}
	}
}

func (s *seeding) sendRootHashMeta(message net.Message) {
	q := new(syncpb.MetaQuery)
	err := proto.Unmarshal(message.Data(), q)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("Fail to unmarshal SyncQuery message.")
		s.netService.ClosePeer(message.MessageFrom(), errors.New("invalid SyncMetaQuery message"))
		return
	}

	logging.Console().WithFields(logrus.Fields{
		"peerID":    message.MessageFrom(),
		"from":      q.From,
		"to":        q.To,
		"chunkSize": q.ChunkSize,
	}).Info("Sync: Seeding manager received hashMeta request.")

	block, err := s.bm.BlockByHeight(q.From)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"height": q.From,
			"err":    err,
		}).Info("Fail to blockByHeight for comparing block hash.")
		return
	}

	if !byteutils.Equal(q.Hash, block.Hash()) {
		logging.WithFields(logrus.Fields{
			"height":           q.From,
			"hashInQuery":      block.Hash(),
			"hashInBlockChain": byteutils.Bytes2Hex(q.Hash),
		}).Info("Block hash is different")
		return
	}

	if err := s.chunkSizeCheck(q.ChunkSize); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to check chunk size.")
		return
	}

	tailHeight := s.bm.TailBlock().Height()
	if q.From+q.ChunkSize-1 > tailHeight || q.To > tailHeight {
		logging.WithFields(logrus.Fields{
			"err":        "request tail height is too high",
			"from":       q.From,
			"to":         q.To,
			"chunkSize":  q.ChunkSize,
			"tailHeight": tailHeight,
			"msgFrom":    message.MessageFrom(),
		}).Info("request tail height is too high")
		return
	}

	numberOfBlocks := q.To - q.From + 1

	if numberOfBlocks > maxNumberOfBlocks {
		logging.WithFields(logrus.Fields{
			"requester":         message.MessageFrom(),
			"from":              q.From,
			"to":                q.To,
			"maxNumberOfBlocks": maxNumberOfBlocks,
		}).Warn("Block is too many to crawling at once")
		return
	}

	if numberOfBlocks < 0 {
		logging.WithFields(logrus.Fields{
			"requester": message.MessageFrom(),
			"from":      q.From,
			"to":        q.To,
		}).Warn("to is lower than from ")
		return
	}

	allHashes := make([][]byte, numberOfBlocks)
	for i := q.From; i <= q.To; i++ {
		block, err = s.bm.BlockByHeight(i)
		if err != nil {
			logging.WithFields(logrus.Fields{
				"height": i,
				"err":    err,
			}).Error("Fail to blockByHeight for gathering canonical chain hashes")
			return
		}

		allHashes[i-q.From] = block.Hash()
	}

	n := numberOfBlocks / q.ChunkSize
	rootHashes := make([][]byte, n)

	for i := uint64(0); i < n; i++ {
		hashes := allHashes[i*q.ChunkSize : (i+1)*q.ChunkSize]
		rootHashes[i] = generateHashTrie(hashes).RootHash()
	}

	meta := new(syncpb.RootHashMeta)
	meta.From = q.From
	meta.ChunkSize = q.ChunkSize
	meta.RootHashes = rootHashes

	sendData, err := proto.Marshal(meta)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to marshal RootHashMeta")
		return
	}

	s.netService.SendMessageToPeer(
		net.SyncMeta,
		sendData,
		net.MessagePriorityLow,
		message.MessageFrom(),
	)

	logging.WithFields(logrus.Fields{
		"numberOfRootHashes": len(meta.RootHashes),
		"meta":               meta,
		"myInfo":             s.netService.Node().ID(),
	}).Info("RootHashMeta response succeeded")
}

func (s *seeding) sendBlockChunk(message net.Message) {
	q := new(syncpb.BlockChunkQuery)
	err := proto.Unmarshal(message.Data(), q)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("Fail to unmarshal BlockChunkQuery message.")
		s.netService.ClosePeer(message.MessageFrom(), errors.New("invalid BlockChunkQuery message"))
		return
	}

	if err := s.chunkSizeCheck(q.ChunkSize); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to check chunk size.")
		return
	}

	tailHeight := s.bm.TailBlock().Height()
	if q.From+q.ChunkSize-1 > tailHeight {
		logging.WithFields(logrus.Fields{
			"err":        "request tail height is too high",
			"from":       q.From,
			"chunkSize":  q.ChunkSize,
			"tailHeight": tailHeight,
			"msgFrom":    message.MessageFrom(),
		}).Info("request tail height is too high")
		return
	}
	pbBlockChunk := make([]*corepb.Block, q.ChunkSize)

	for i := uint64(0); i < q.ChunkSize; i++ {
		block, err := s.bm.BlockByHeight(i + q.From)
		if err != nil {
			logging.WithFields(logrus.Fields{
				"height": i,
				"err":    err,
			}).Error("Fail to blockByHeight for make blockChunk")
			return
		}

		pbBlock, err := block.ToProto()
		if err != nil {
			logging.Error("Fail to convert block to pbBlock")
			return
		}
		pbBlockChunk[i] = pbBlock.(*corepb.Block)
	}

	data := new(syncpb.BlockChunk)
	data.From = q.From
	data.Blocks = pbBlockChunk

	sendData, err := proto.Marshal(data)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to marshal BlockChunk")
		return
	}

	s.netService.SendMessageToPeer(
		net.SyncBlockChunk,
		sendData,
		net.MessagePriorityLow,
		message.MessageFrom(),
	)

	logging.WithFields(logrus.Fields{
		"from":             data.From,
		"Number of Blocks": len(data.Blocks),
	}).Info("BlockChunk response succeeded")

}

func (s *seeding) chunkSizeCheck(n uint64) error {
	if n < s.minChunkSize {
		return errors.New("ChunkSize is too small")
	}
	if n > s.maxChunkSize {
		return errors.New("ChunkSize is too large")
	}
	return nil
}
