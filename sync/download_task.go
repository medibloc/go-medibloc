package sync

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type downloadTask struct {
	netService          net.Service
	query               []byte
	from                uint64
	chunkSize           uint64
	peers               map[string]struct{}
	rootHash            common.Hash
	blocks              []*core.BlockData
	createdTime         time.Time
	startTime           time.Time
	endTime             time.Time
	blockChunkMessageCh chan net.Message
	quitCh              chan bool
	doneCh              chan *downloadTask
	pid                 string
}

func newDownloadTask(netService net.Service, peers map[string]struct{}, from uint64, chunkSize uint64, rootHash common.Hash, doneCh chan *downloadTask) *downloadTask {
	return &downloadTask{
		netService:          netService,
		query:               nil,
		from:                from,
		chunkSize:           chunkSize,
		peers:               peers,
		rootHash:            rootHash,
		blocks:              nil,
		createdTime:         time.Now(),
		startTime:           time.Time{},
		endTime:             time.Time{},
		blockChunkMessageCh: make(chan net.Message, 1),
		quitCh:              make(chan bool, 2),
		doneCh:              doneCh,
		pid:                 "",
	}
}

func (dt *downloadTask) start() {
	dt.generateBlockChunkQuery()
	dt.sendBlockChunkRequest()
	dt.startTime = time.Now()
	go dt.startLoop()
}

func (dt *downloadTask) stop() {
	dt.quitCh <- true
}

func (dt *downloadTask) startLoop() {
	timerChan := time.NewTicker(time.Second * 3).C //TODO: set retry time
	for {
		select {
		case <-timerChan:
			dt.sendBlockChunkRequest()
		case blockChunkMessage := <-dt.blockChunkMessageCh:
			dt.verifyBlockChunkMessage(blockChunkMessage)
		case <-dt.quitCh:
			dt.doneCh <- dt
			return
		}
	}
}

func (dt *downloadTask) verifyBlockChunkMessage(message net.Message) {

	blockChunk := new(syncpb.BlockChunk)
	err := proto.Unmarshal(message.Data(), blockChunk)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("Fail to unmarshal HashMeta message.")
		dt.netService.ClosePeer(message.MessageFrom(), errors.New("invalid blockChunk message"))
		return
	}

	if uint64(len(blockChunk.Blocks)) != dt.chunkSize {
		logging.WithFields(logrus.Fields{
			"err":                "block chunksize is unmatched",
			"chunkSize":          dt.chunkSize,
			"received chunkSize": len(blockChunk.Blocks),
			"msgFrom":            message.MessageFrom(),
		}).Warn("block chunksize is unmatched")
		dt.netService.ClosePeer(message.MessageFrom(), errors.New("block chunksize is unmatched"))
		return
	}

	if blockChunk.From != dt.from {
		logging.WithFields(logrus.Fields{
			"err":           "block range is unmatched",
			"from":          dt.from,
			"received from": blockChunk.From,
			"msgFrom":       message.MessageFrom(),
		}).Warn("block range is unmatched")
		dt.netService.ClosePeer(message.MessageFrom(), errors.New("block range is unmatched"))
		return
	}

	var downloadedHashes []common.Hash
	blocks := make([]*core.BlockData, 0, dt.chunkSize)
	for _, pbBlock := range blockChunk.Blocks {
		block := new(core.BlockData)
		block.FromProto(pbBlock)
		blocks = append(blocks, block)

		if err := block.VerifyIntegrity(); err != nil {
			logging.WithFields(logrus.Fields{
				"Block Height": block.Height(),
				"err":          err,
				"msgFrom":      message.MessageFrom(),
			}).Warn("Fail to verify block integrity.")
			return
		}
		downloadedHashes = append(downloadedHashes, block.Hash())
	}

	rootHash := common.BytesToHash(generateHashTrie(downloadedHashes).RootHash())
	if rootHash != dt.rootHash {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("BlockChunks root hash is not matched.")
		return
	}

	dt.blocks = blocks
	dt.endTime = time.Now()
	dt.pid = message.MessageFrom()
	dt.stop()
}

func (dt *downloadTask) removePeer(peer string, errMsg string) {
	delete(dt.peers, peer)
	dt.netService.ClosePeer(peer, errors.New(errMsg))
}

func (dt *downloadTask) generateBlockChunkQuery() {
	q := &syncpb.BlockChunkQuery{
		From:      dt.from,
		ChunkSize: dt.chunkSize,
	}
	query, err := proto.Marshal(q)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to marshal BlockChunkQuery")
		return
	}
	dt.query = query
}

func (dt *downloadTask) sendBlockChunkRequest() {
	randomIndex := rand.Intn(len(dt.peers))
	var randomPeer string
	index := 0
	for peer := range dt.peers {
		if index == randomIndex {
			randomPeer = peer
			break
		}
		index++
	}
	dt.netService.SendMessageToPeer(net.SyncBlockChunkRequest, dt.query, net.MessagePriorityLow, randomPeer)
	logging.Console().WithFields(logrus.Fields{
		"block from (height)": dt.from,
		"to (peerID)":         randomPeer,
		"nPeers":              len(dt.peers),
		"peers":               dt.peers,
	}).Info("BlockChunkRequest is sent")
}

func (dt *downloadTask) String() string {
	return fmt.Sprintf("<DownloadTask>From:%v,", dt.from)
}
