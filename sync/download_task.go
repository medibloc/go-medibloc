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
	"fmt"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

//ErrFailVerification is occurred when received block chunk failed to verify.
var ErrFailVerification = errors.New("fail to verification block chunk")

type downloadTask struct {
	netService          net.Service
	query               []byte
	from                uint64
	chunkSize           uint64
	peers               map[string]struct{}
	rootHash            string
	blocks              []*core.BlockData
	createdTime         time.Time
	startTime           time.Time
	endTime             time.Time
	blockChunkMessageCh chan net.Message
	quitCh              chan bool
	pid                 string
	interval            time.Duration
}

func newDownloadTask(netService net.Service, peers map[string]struct{}, from uint64, chunkSize uint64, rootHash string, interval time.Duration) *downloadTask {
	dt := &downloadTask{
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
		pid:                 "",
		interval:            interval,
	}
	dt.generateBlockChunkQuery()
	dt.startTime = time.Now().Add(-1 * interval)
	return dt
}

func (dt *downloadTask) verifyBlockChunkMessage(message net.Message) error {
	if dt.blocks != nil {
		logging.Console().Infof("Block Chunk is already received from:%v", dt.from)
		return ErrFailVerification
	}

	blockChunk := new(syncpb.BlockChunk)
	err := proto.Unmarshal(message.Data(), blockChunk)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("Fail to unmarshal HashMeta message.")
		dt.netService.ClosePeer(message.MessageFrom(), errors.New("invalid blockChunk message"))
		return ErrFailVerification
	}

	if uint64(len(blockChunk.Blocks)) != dt.chunkSize {
		logging.WithFields(logrus.Fields{
			"err":                "block chunksize is unmatched",
			"chunkSize":          dt.chunkSize,
			"received chunkSize": len(blockChunk.Blocks),
			"msgFrom":            message.MessageFrom(),
		}).Warn("block chunksize is unmatched")
		dt.netService.ClosePeer(message.MessageFrom(), errors.New("block chunksize is unmatched"))
		return ErrFailVerification
	}

	if blockChunk.From != dt.from {
		logging.WithFields(logrus.Fields{
			"err":           "block range is unmatched",
			"from":          dt.from,
			"received from": blockChunk.From,
			"msgFrom":       message.MessageFrom(),
		}).Warn("block range is unmatched")
		dt.netService.ClosePeer(message.MessageFrom(), errors.New("block range is unmatched"))
		return ErrFailVerification
	}

	var downloadedHashes [][]byte
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
			return ErrFailVerification
		}
		downloadedHashes = append(downloadedHashes, block.Hash())
	}

	rootHash := generateHashTrie(downloadedHashes).RootHash()
	if !byteutils.Equal(rootHash, byteutils.Hex2Bytes(dt.rootHash)) {
		logging.WithFields(logrus.Fields{
			"err":         err,
			"rootHash":    byteutils.Bytes2Hex(rootHash),
			"dt.rootHash": dt.rootHash,
			"msgFrom":     message.MessageFrom(),
		}).Error("BlockChunks root hash is not matched.")
		return ErrFailVerification
	}

	dt.blocks = blocks
	dt.endTime = time.Now()
	dt.pid = message.MessageFrom()
	return nil
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
		}).Error("Failed to marshal BlockChunkQuery")
		return
	}
	dt.query = query
}

func (dt *downloadTask) sendBlockChunkRequest() {
	if time.Now().Sub(dt.startTime) < dt.interval {
		return
	}
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
	dt.startTime = time.Now()
	dt.netService.SendMessageToPeer(net.SyncBlockChunkRequest, dt.query, net.MessagePriorityLow, randomPeer)
	logging.Console().WithFields(logrus.Fields{
		"block from (height)": dt.from,
		"to (peerID)":         randomPeer,
		"nPeers":              len(dt.peers),
		"peers":               dt.peers,
	}).Info("BlockChunkRequest is sent")
}

//String return stringified downloadTask
func (dt *downloadTask) String() string {
	return fmt.Sprintf("<DownloadTask from:%v>", dt.from)
}
