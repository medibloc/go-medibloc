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
	"math"
	"sort"
	"strings"
	"time"

	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type download struct {
	netService net.Service
	bm         BlockManager
	messageCh  chan net.Message
	quitCh     chan bool

	mu        sync.Mutex
	activated bool

	downloadStart     bool
	from              uint64
	chunkSize         uint64
	pidRootHashesMap  map[string][]string
	rootHashPIDsMap   map[string]map[string]struct{}
	taskQueue         []*downloadTask
	runningTasks      map[uint64]*downloadTask
	finishedTasks     *taskList
	taskAddCh         chan *downloadTask
	maxRunningTask    uint32
	chunkCacheSize    uint64
	minimumPeersCount uint32
	respondingPeers   map[string]struct{}
	interval          time.Duration
	timeout           time.Duration
}

//IsActivated return status of activation
func (d *download) IsActivated() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.activated
}

func newDownload(config *medletpb.SyncConfig) *download {
	return &download{
		netService:        nil,
		bm:                nil,
		messageCh:         make(chan net.Message, 128),
		quitCh:            make(chan bool, 1),
		activated:         false,
		downloadStart:     false,
		from:              0,
		chunkSize:         config.DownloadChunkSize,
		pidRootHashesMap:  make(map[string][]string),
		rootHashPIDsMap:   make(map[string]map[string]struct{}),
		taskQueue:         make([]*downloadTask, 0),
		runningTasks:      make(map[uint64]*downloadTask),
		finishedTasks:     nil,
		taskAddCh:         make(chan *downloadTask, 2),
		maxRunningTask:    config.DownloadMaxConcurrentTasks,
		chunkCacheSize:    config.DownloadChunkCacheSize,
		minimumPeersCount: config.MinimumPeers,
		respondingPeers:   make(map[string]struct{}),
		interval:          time.Duration(config.RequestInterval) * time.Second,
		timeout:           time.Duration(config.FinisherTimeout) * time.Second,
	}
}

func (d *download) setup(netService net.Service, bm BlockManager) {
	d.netService = netService
	d.bm = bm
}

func (d *download) start() {
	d.netService.Register(net.NewSubscriber(d, d.messageCh, false, net.SyncMeta, net.MessageWeightZero))
	d.netService.Register(net.NewSubscriber(d, d.messageCh, false, net.SyncBlockChunk, net.MessageWeightZero))

	d.from = d.bm.LIB().Height()
	d.finishedTasks = &taskList{
		tasks:     make([]*downloadTask, 0),
		offset:    d.from,
		chunkSize: d.chunkSize,
	}
	d.sendMetaQuery()

	d.mu.Lock()
	d.activated = true
	d.mu.Unlock()

	go d.subscribeLoop()
}

func (d *download) stop() {
	d.quitCh <- true
}

func (d *download) subscribeLoop() {
	defer d.deregisterSubscriber()

	logging.Console().Info("Sync: Download manager is started.")

	intervalTicker := time.NewTicker(d.interval)
	timeoutTimerStart := time.Now()
	timeoutTimer := time.NewTimer(d.timeout)
	prevEstablishedPeersCount := int32(0)

	for {
		select {
		case <-intervalTicker.C:
			hasWork := false
			if prevEstablishedPeersCount < d.netService.Node().EstablishedPeersCount() {
				hasWork = true
				d.sendMetaQuery()
				prevEstablishedPeersCount = d.netService.Node().EstablishedPeersCount()
			}
			logging.Console().WithFields(logrus.Fields{
				"taskQueue":         d.taskQueue,
				"runningTasks":      d.runningTasks,
				"finishedTasks":     d.finishedTasks,
				"currentTailHeight": d.bm.TailBlock().Height(),
				"timeout":           time.Now().Sub(timeoutTimerStart),
			}).Info("Sync: download service status")

			if len(d.runningTasks) > 0 {
				hasWork = true
				for _, t := range d.runningTasks {
					t.sendBlockChunkRequest()
				}
			}
			if !hasWork {
				continue
			}

		case <-timeoutTimer.C:
			logging.Console().WithFields(logrus.Fields{
				"from": d.from,
				"to":   d.bm.TailBlock().Height(),
			}).Info("Sync: Download manager is stopped by timeout")
			return
		case <-d.quitCh:
			logging.Console().WithFields(logrus.Fields{
				"from": d.from,
				"to":   d.bm.TailBlock().Height(),
			}).Info("Sync: Download manager is stopped.")
			return
		case message := <-d.messageCh:
			switch message.MessageType() {
			case net.SyncMeta:
				d.updateMeta(message)
			case net.SyncBlockChunk:
				d.findTaskForBlockChunk(message)
			}
		}
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}
		timeoutTimerStart = time.Now()
		timeoutTimer.Reset(d.timeout)
	}
}

func (d *download) runNextTask() {
	if d.finishedTasks.CacheSize() > d.chunkCacheSize {
		logging.Console().WithFields(logrus.Fields{
			"cached finished task": d.finishedTasks.CacheSize(),
			"cache size":           d.chunkCacheSize,
		}).Info("Sync: Download CacheSize limited. Waiting for finish previous task.")
		return
	}
	count := d.chunkCacheSize - d.finishedTasks.CacheSize()
	for {
		if len(d.runningTasks) >= int(d.maxRunningTask) {
			break
		}
		if len(d.taskQueue) < 1 {
			break
		}
		if count < 1 {
			break
		}
		count--

		t := d.taskQueue[0]
		d.runningTasks[t.from] = t
		d.taskQueue = d.taskQueue[1:]
		t.sendBlockChunkRequest()
	}
}

func (d *download) updateMeta(message net.Message) {
	rootHashMeta := new(syncpb.RootHashMeta)
	err := proto.Unmarshal(message.Data(), rootHashMeta)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("Fail to unmarshal HashMeta message.")
		d.netService.ClosePeer(message.MessageFrom(), errors.New("invalid HashMeta message"))
		return
	}

	if rootHashMeta.From != d.from || rootHashMeta.ChunkSize != d.chunkSize {
		logging.WithFields(logrus.Fields{
			"From":             rootHashMeta.From,
			"ChunkSize":        rootHashMeta.ChunkSize,
			"number of Hashes": len(rootHashMeta.RootHashes),
			"err":              "From or ChunkSize is unmatched",
			"msgFrom":          message.MessageFrom(),
		}).Warn("From or ChunkSize is unmatched")
		return
	}

	d.respondingPeers[message.MessageFrom()] = struct{}{}
	d.setPIDRootHashesMap(message.MessageFrom(), rootHashMeta.RootHashes)
	d.setRootHashPIDsMap(message.MessageFrom(), rootHashMeta.RootHashes)
	d.checkMajorMeta()
	logging.Infof("RootHash Meta is updated. (%v/%v)", len(d.pidRootHashesMap), d.netService.Node().PeersCount())
}

func (d *download) setPIDRootHashesMap(pid string, rootHashesByte [][]byte) {
	rootHashes := make([]string, len(rootHashesByte))
	for i, rootHash := range rootHashesByte {
		rootHashes[i] = byteutils.Bytes2Hex(rootHash)
	}
	d.pidRootHashesMap[pid] = rootHashes
}

func (d *download) setRootHashPIDsMap(pid string, rootHashesByte [][]byte) {
	for _, rootHash := range rootHashesByte {
		rootHashHex := byteutils.Bytes2Hex(rootHash)
		if _, ok := d.rootHashPIDsMap[rootHashHex]; ok == false {
			d.rootHashPIDsMap[rootHashHex] = make(map[string]struct{})
		}
		d.rootHashPIDsMap[rootHashHex][pid] = struct{}{}
	}
}

func (d *download) checkMajorMeta() {
	if !d.majorityCheck(len(d.pidRootHashesMap)) {
		return
	}
	i := len(d.runningTasks) + d.finishedTasks.Len() + len(d.taskQueue)
	for {
		peerCounter := make(map[string]int)
		for _, rootHashes := range d.pidRootHashesMap {
			if len(rootHashes) > i {
				peerCounter[rootHashes[i]]++
			}
		}
		majorNotFound := true
		for rootHashHex, nPeers := range peerCounter {
			if d.majorityCheck(nPeers) {
				logging.Infof("Major RootHash was found from %v", d.from+uint64(i)*d.chunkSize)
				//createDownloadTask
				majorNotFound = false
				t := newDownloadTask(d.netService, d.rootHashPIDsMap[rootHashHex], d.from+uint64(i)*d.chunkSize, d.chunkSize, rootHashHex, d.interval)
				d.taskQueue = append(d.taskQueue, t)
				d.runNextTask()
				break
			}
		}
		if majorNotFound {
			logging.Infof("Major RootHash was not found at %v", d.from+uint64(i)*d.chunkSize)
			break
		}
		i++
	}
}

func (d *download) findTaskForBlockChunk(message net.Message) {

	blockChunk := new(syncpb.BlockChunk)
	err := proto.Unmarshal(message.Data(), blockChunk)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgFrom": message.MessageFrom(),
		}).Warn("Fail to unmarshal HashMeta message.")
		d.netService.ClosePeer(message.MessageFrom(), errors.New("invalid blockChunk message"))
		return
	}

	if t, ok := d.runningTasks[blockChunk.From]; ok {
		err := t.verifyBlockChunkMessage(message)
		if err != nil {
			t.sendBlockChunkRequest()
			return
		}
		delete(d.runningTasks, t.from)
		d.finishedTasks.Add(t)
		if err := d.pushBlockDataChunk(); err != nil {
			logging.Console().Error("PushBlockDataChunk Failed", err)
		}
		if len(d.taskQueue) > 0 {
			d.runNextTask()
		} else if len(d.runningTasks) == 0 {
			logging.Info("No more task in queue and running Tasks")
		}
	}
}

func (d *download) sendMetaQuery() error {
	mq := new(syncpb.MetaQuery)
	mq.From = d.from
	mq.Hash = d.bm.LIB().Hash()
	mq.ChunkSize = d.chunkSize

	sendData, err := proto.Marshal(mq)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to marshal MetaQuery")
		return err
	}
	filter := new(net.ChainSyncPeersFilter)
	filter.SetExcludedPIDs(d.respondingPeers)
	peers := d.netService.SendMessageToPeers(net.SyncMetaRequest, sendData, net.MessagePriorityLow, filter)
	logging.Console().WithFields(logrus.Fields{
		"mq":                       mq,
		"peers":                    peers,
		"sendData":                 sendData,
		"numberOfPeers":            d.netService.Node().PeersCount(),
		"numberOfEstablishedPeers": d.netService.Node().EstablishedPeersCount(),
	}).Info("Sync Meta Request was sent")
	return nil
}

func (d *download) pushBlockDataChunk() error {
	for {
		task := d.finishedTasks.Next()
		if task == nil {
			break
		}
		blocks := task.blocks
		for _, b := range blocks {
			err := d.bm.PushBlockData(b)
			if err == core.ErrDuplicatedBlock {
				logging.Console().WithFields(logrus.Fields{
					"hash":   b.Hash(),
					"height": b.Height(),
				}).Infof("Block height (%d) is already pushed", b.Height())
				continue
			}
			if err != nil {
				return err
			}
		}
		logging.Console().WithFields(logrus.Fields{
			"taskFrom": task.from,
		}).Infof("Pushing blockChunk from %d is completed!!", task.from)
	}

	return nil
}

func (d *download) majorityCheck(n int) bool {
	numberOfPeers := float64(d.netService.Node().EstablishedPeersCount())
	majorTh := int(math.Ceil(numberOfPeers / 2.0))
	if n < majorTh || n < int(d.minimumPeersCount) {
		return false
	}
	return true
}

func (d *download) flush() {
	d.pidRootHashesMap = make(map[string][]string)
	d.rootHashPIDsMap = make(map[string]map[string]struct{})
	d.taskQueue = make([]*downloadTask, 0)
	d.runningTasks = make(map[uint64]*downloadTask)
	d.respondingPeers = make(map[string]struct{})
}

func (d *download) deregisterSubscriber() {
	d.mu.Lock()
	d.activated = false
	d.mu.Unlock()
	d.flush()

	d.netService.Deregister(net.NewSubscriber(d, d.messageCh, false, net.SyncMeta, net.MessageWeightZero))
	d.netService.Deregister(net.NewSubscriber(d, d.messageCh, false, net.SyncBlockChunk, net.MessageWeightZero))
}

// taskList manages finished tasks.
type taskList struct {
	tasks     []*downloadTask
	offset    uint64
	chunkSize uint64
}

func (l taskList) Len() int {
	return len(l.tasks)
}

func (l taskList) Less(i, j int) bool {
	return l.tasks[i].from < l.tasks[j].from
}

func (l taskList) Swap(i, j int) {
	l.tasks[i], l.tasks[j] = l.tasks[j], l.tasks[i]
}

func (l *taskList) Add(task *downloadTask) {
	l.tasks = append(l.tasks, task)
	sort.Sort(l)
}

func (l *taskList) Next() *downloadTask {
	for _, task := range l.tasks {
		if task.from > l.offset {
			return nil
		}
		if task.from == l.offset {
			l.offset += l.chunkSize
			return task
		}
	}
	return nil
}

func (l *taskList) CacheSize() uint64 {
	for i, task := range l.tasks {
		if task.from > l.offset {
			return uint64(len(l.tasks) - i + 1)
		}
		if task.from == l.offset {
			return uint64(len(l.tasks) - i)
		}
	}
	return 0
}

func (l *taskList) String() string {
	var s []string
	for _, task := range l.tasks {
		s = append(s, task.String())
	}
	return strings.Join(s, ",")
}
