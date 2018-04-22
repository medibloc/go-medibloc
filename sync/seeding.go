package sync

import (
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/gogo/protobuf/proto"
	"errors"
	"github.com/medibloc/go-medibloc/core/pb"
)

type seeding struct {
	netService net.Service
	bm         BlockManager

	quitCh    chan bool
	messageCh chan net.Message
}

func newSeeding(netService net.Service, bm BlockManager) *seeding {
	return &seeding{
		netService: netService,
		bm:         bm,
		quitCh:     make(chan bool, 2),
		messageCh:  make(chan net.Message, 128),
	}
}

func (s *seeding) start() {
	s.netService.Register(net.NewSubscriber(s, s.messageCh, false, net.SyncMetaRequest, net.MessageWeightZero))
	s.netService.Register(net.NewSubscriber(s, s.messageCh, false, net.SyncBlockChunkRequest, net.MessageWeightZero))
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
			logging.Console().Info("Sync: seeding Service Stopped")
			return
		case message := <-s.messageCh:
			switch message.MessageType() {
			case net.SyncMetaRequest:
				s.sendRootHashMeta(message)
			case net.SyncBlockChunkRequest:
				s.sendBlockChunk(message)
			}
		}
	}
}

func (s *seeding) sendRootHashMeta(message net.Message) {
	q := new(syncpb.MetaQuery)
	err := proto.Unmarshal(message.Data(), q)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("Fail to unmarshal SyncQuery message.")
		s.netService.ClosePeer(message.MessageFrom(), errors.New("invalid SyncMetaQuery message"))
		return
	}

	if common.BytesToHash(q.Hash) != s.bm.BlockByHeight(q.From).Hash() {
		logging.WithFields(logrus.Fields{
			"err": "Meta Query Request Denied.",
		}).Info("Meta Query Request Denied.")
		return
	}

	tailHeight := s.bm.TailBlock().Height()
	if q.From+q.ChunkSize < tailHeight {
		logging.WithFields(logrus.Fields{
			"err":        "request tail height is too high",
			"from":       q.From,
			"chunkSize":  q.ChunkSize,
			"tailHeight": tailHeight,
			"msgFrom":    message.MessageFrom(),
		}).Info("request tail height is too high")
		return
	}

	allHashes := make([]common.Hash, tailHeight-q.From)
	for i := q.From; i < tailHeight; i++ {
		allHashes[i] = s.bm.BlockByHeight(i).Hash()
	}

	n := tailHeight / q.ChunkSize
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
		}).Debug("Failed to marshal RootHashMeta")
		return
	}

	s.netService.SendMessageToPeer(
		net.SyncMeta,
		sendData,
		net.MessagePriorityLow,
		message.MessageFrom(),
	)

	logging.Info("RootHashMeta response succeeded")
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

	tailHeight := s.bm.TailBlock().Height()
	if q.From+q.ChunkSize < tailHeight {
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

	for i := q.From; i < q.From+q.ChunkSize; i++ {
		pbBlock, err := s.bm.BlockByHeight(i).ToProto()
		if err != nil {
			logging.Error("Fail to convert block to pbBlock")
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
		}).Debug("Failed to marshal BlockChunk")
		return
	}

	s.netService.SendMessageToPeer(
		net.SyncMeta,
		sendData,
		net.MessagePriorityLow,
		message.MessageFrom(),
	)

	logging.Info("BlockChunk response succeeded")
}
