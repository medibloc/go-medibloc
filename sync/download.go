package sync

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

type download struct {
	netService net.Service
	bm         BlockManager
	messageCh  chan net.Message
	quitCh     chan bool

	activated        bool
	downloadStart    bool
	from             uint64
	chunkSize        uint64
	pidRootHashesMap map[string][]common.Hash
	rootHashPIDsMap  map[common.Hash]map[string]struct{}
	taskQueue        []*downloadTask
	runningTasks     map[uint64]*downloadTask
	finishedTasks    *taskList
	taskAddCh        chan *downloadTask
	taskDoneCh       chan *downloadTask
	maxRunningTask   uint32
	chunkCacheSize   uint64
}

func newDownload(config *medletpb.SyncConfig) *download {
	return &download{
		netService:       nil,
		bm:               nil,
		messageCh:        make(chan net.Message, 128),
		quitCh:           make(chan bool, 1),
		activated:        false,
		downloadStart:    false,
		from:             0,
		chunkSize:        config.DownloadChunkSize,
		pidRootHashesMap: make(map[string][]common.Hash),
		rootHashPIDsMap:  make(map[common.Hash]map[string]struct{}),
		taskQueue:        make([]*downloadTask, 0),
		runningTasks:     make(map[uint64]*downloadTask),
		finishedTasks:    nil,
		taskAddCh:        make(chan *downloadTask, 2),
		taskDoneCh:       make(chan *downloadTask),
		maxRunningTask:   config.DownloadMaxConcurrentTasks,
		chunkCacheSize:   config.DownloadChunkCacheSize,
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
	go d.subscribeLoop()

}

func (d *download) stop() {
	d.netService.Deregister(net.NewSubscriber(d, d.messageCh, false, net.SyncMeta, net.MessageWeightZero))
	d.netService.Deregister(net.NewSubscriber(d, d.messageCh, false, net.SyncBlockChunk, net.MessageWeightZero))

	d.quitCh <- true
}

func (d *download) subscribeLoop() {
	timerChan := time.NewTicker(time.Second * 5).C //TODO: set timeout
	retryMetaRequestCnt := 0
	for {
		select {
		case <-timerChan:
			if !d.majorityCheck(len(d.pidRootHashesMap)) {
				d.sendMetaQuery()
				retryMetaRequestCnt++
			}
			logging.WithFields(logrus.Fields{
				"taskQueue":         d.taskQueue,
				"runningTasks":      d.runningTasks,
				"finishedTasks":     d.finishedTasks,
				"currentTailHeight": d.bm.TailBlock().Height(),
			}).Info("Sync: download service status")

		case <-d.quitCh:
			logging.Console().Info("Sync: download Service Stopped", d.bm.TailBlock().Height())
			return
		case message := <-d.messageCh:
			switch message.MessageType() {
			case net.SyncMeta:
				d.updateMeta(message)
			case net.SyncBlockChunk:
				d.findTaskForBlockChunk(message)

			}
		case t := <-d.taskDoneCh:
			delete(d.runningTasks, t.from)
			d.finishedTasks.Add(t)
			if err := d.pushBlockDataChunk(); err != nil {
				logging.Console().Infof("PushBlockDataChunk Failed", err)
			}
			if len(d.taskQueue) > 0 {
				d.runNextTask()
			} else if len(d.runningTasks) == 0 {
				d.downloadFinishCheck()
			}
		}
	}
}

func (d *download) runNextTask() {
	if d.finishedTasks.CacheSize() > d.chunkCacheSize {
		logging.Console().WithFields(logrus.Fields{
			"len(d.finishedTasks)":  d.finishedTasks.CacheSize(),
			"int(d.chunkCacheSize)": int(d.chunkCacheSize),
		}).Info("CacheSize limited")
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
		t.start()
	}
}

func (d *download) updateMeta(message net.Message) {
	logging.Info("RootHash Meta Received")

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

	d.setPIDRootHashesMap(message.MessageFrom(), rootHashMeta.RootHashes)
	d.setRootHashPIDsMap(message.MessageFrom(), rootHashMeta.RootHashes)
	d.checkMajorMeta()
	logging.Info("Received rootHash Meta is updated")
}

func (d *download) setPIDRootHashesMap(pid string, rootHashesByte [][]byte) {
	rootHashes := make([]common.Hash, len(rootHashesByte))
	for i, rootHash := range rootHashesByte {
		rootHashes[i] = common.BytesToHash(rootHash)
	}
	d.pidRootHashesMap[pid] = rootHashes
}

func (d *download) setRootHashPIDsMap(pid string, rootHashesByte [][]byte) {
	for _, rootHashByte := range rootHashesByte {
		rootHash := common.BytesToHash(rootHashByte)
		if _, ok := d.rootHashPIDsMap[rootHash]; ok == false {
			d.rootHashPIDsMap[rootHash] = make(map[string]struct{})
		}
		d.rootHashPIDsMap[rootHash][pid] = struct{}{}
	}
}

func (d *download) checkMajorMeta() {
	if !d.majorityCheck(len(d.pidRootHashesMap)) {
		return
	}
	i := len(d.runningTasks) + d.finishedTasks.Len() + len(d.taskQueue)
	for {
		peerCounter := make(map[common.Hash]int)
		for _, rootHashes := range d.pidRootHashesMap {
			if len(rootHashes) > i {
				peerCounter[rootHashes[i]]++
			}
		}
		majorNotFound := true
		for rootHash, nPeers := range peerCounter {
			if d.majorityCheck(nPeers) {
				logging.Infof("Major RootHash was found from %v", d.from+uint64(i)*d.chunkSize)
				//createDownloadTask
				majorNotFound = false
				t := newDownloadTask(d.netService, d.rootHashPIDsMap[rootHash], d.from+uint64(i)*d.chunkSize, d.chunkSize, rootHash, d.taskDoneCh)
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
		//d.netService.ClosePeer(message.MessageFrom(), errors.New("invalid blockChunk message"))
		return
	}

	if t, ok := d.runningTasks[blockChunk.From]; ok {
		t.blockChunkMessageCh <- message
	}
}

func (d *download) sendMetaQuery() error {
	mq := new(syncpb.MetaQuery)
	mq.From = d.from
	mq.Hash = d.bm.LIB().Hash().Bytes()
	mq.ChunkSize = d.chunkSize

	sendData, err := proto.Marshal(mq)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to marshal MetaQuery")
		return err
	}
	d.netService.SendMessageToPeers(net.SyncMetaRequest, sendData, net.MessagePriorityLow, new(net.ChainSyncPeersFilter))
	logging.WithFields(logrus.Fields{
		"mq":       mq,
		"sendData": sendData,
	}).Info("Sync Meta Request was sent")
	return nil
}

func (d *download) downloadFinishCheck() {
	logging.Console().Debug("finished task:", d.finishedTasks)
	lastFinishedTask := d.finishedTasks.tasks[len(d.finishedTasks.tasks)-1]
	if len(d.pidRootHashesMap[lastFinishedTask.pid]) > len(d.finishedTasks.tasks) {
		return
	}

	if d.finishedTasks.CacheSize() > 0 {
		return
	}
	logging.WithFields(logrus.Fields{
		"height": d.bm.TailBlock().Height(),
	}).Info("Sync Service Download complete")
	d.quitCh <- true
}

func (d *download) pushBlockDataChunk() error {
	for {
		task := d.finishedTasks.Next()
		if task == nil {
			return nil
		}
		blocks := task.blocks
		for _, b := range blocks {
			if d.bm.BlockByHash(b.Hash()) != nil {
				continue
			}
			if err := d.bm.PushBlockData(b); err != nil {
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
	numberOfPeers := float64(d.netService.Node().PeersCount())
	majorTh := int(math.Ceil(numberOfPeers / 2.0))
	if n < majorTh {
		return false
	}
	return true
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
