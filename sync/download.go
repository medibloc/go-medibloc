package sync

import (
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/medibloc/go-medibloc/sync/pb"
	"github.com/sirupsen/logrus"
	"github.com/gogo/protobuf/proto"
	"errors"
	"github.com/medibloc/go-medibloc/common"
)

type Download struct {
	netService     net.Service
	bm             BlockManager
	messageCh      chan net.Message
	quitCh         chan bool
	quitTaskLoopCh chan bool

	activated               bool
	downloadStart           bool
	from                    uint64
	chunkSize               uint64
	pidRootHashesMap        map[string][]common.Hash
	rootHashPIDsMap         map[common.Hash]map[string]struct{}
	majorityThreshold       int
	taskQue                 []*DownloadTask
	runningTasks            []*DownloadTask
	finishedTasks           []*DownloadTask
	taskAddCh               chan *DownloadTask
	taskDoneCh              chan *DownloadTask
	recentFinishedTaskIndex uint64
	recentStartedTaskIndex  uint64
	maxRunningTask          uint64
}

func newDownload(netService net.Service, bm BlockManager) *Download {
	return &Download{
		netService:              netService,
		bm:                      bm,
		messageCh:               make(chan net.Message, 128),
		quitCh:                  make(chan bool, 1),
		quitTaskLoopCh:          make(chan bool, 1),
		activated:               false,
		downloadStart:           false,
		from:                    0,
		chunkSize:               10, //TODO: chunksize 주는 법. config? cli?
		pidRootHashesMap:        make(map[string][]common.Hash),
		rootHashPIDsMap:         make(map[common.Hash]map[string]struct{}),
		majorityThreshold:       11, //TODO: 다수의 기준 config? peers?
		taskQue:                 make([]*DownloadTask, 0),
		runningTasks:            make([]*DownloadTask, 0),
		finishedTasks:           make([]*DownloadTask, 0),
		taskAddCh:               make(chan *DownloadTask, 2),
		taskDoneCh:              make(chan *DownloadTask),
		recentFinishedTaskIndex: 0,
		recentStartedTaskIndex:  0,
		maxRunningTask:          5, //TODO: maxTask
	}
}

func (d *Download) setup(chunkSize uint64) {
	d.from = d.bm.LIB().Height()
	d.chunkSize = chunkSize
}

func (d *Download) start() {

	d.netService.Register(net.NewSubscriber(d, d.messageCh, false, net.SyncMeta, net.MessageWeightZero))
	d.netService.Register(net.NewSubscriber(d, d.messageCh, false, net.SyncBlockChunk, net.MessageWeightZero))

	go d.subscribeLoop()
	go d.taskLoop()

}

func (d *Download) Stop() {
	d.netService.Deregister(net.NewSubscriber(d, d.messageCh, false, net.SyncMeta, net.MessageWeightZero))
	d.netService.Deregister(net.NewSubscriber(d, d.messageCh, false, net.SyncBlockChunk, net.MessageWeightZero))

	d.quitCh <- true
}

func (d *Download) subscribeLoop() {
	for {
		select {
		case <-d.quitCh:
			logging.Console().Info("Sync: Download Service Stopped")
			d.quitTaskLoopCh <- true
			return
		case message := <-d.messageCh:
			switch message.MessageType() {
			case net.SyncMeta:
				d.updateMeta(message)
			case net.SyncBlockChunk:
				d.findTaskForBlockChunk(message)
			}
		}
	}
}

func (d *Download) taskLoop() {
	for {
		select {
		case <-d.quitTaskLoopCh:
			return
		case t := <-d.taskAddCh:
			addTask(d.taskQue, t)
			d.runNextTask()
		case t := <-d.taskDoneCh:
			addTask(d.finishedTasks, t)
			removeTask(d.runningTasks, t)
			d.runNextTask()
		}
	}
}

func (d *Download) runNextTask() {
	if len(d.runningTasks) < int(d.maxRunningTask) {
		t := d.taskQue[0]
		addTask(d.runningTasks, t)
		removeTask(d.taskQue, t)
		t.start()
	}
}

func (d *Download) updateMeta(message net.Message) {
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
			"err":     "From or ChunkSize is unmatched",
			"msgFrom": message.MessageFrom(),
		}).Warn("From or ChunkSize is unmatched")
		return
	}

	d.setPIDRootHashesMap(message.MessageFrom(), rootHashMeta.RootHashes)
	d.setRootHashPIDsMap(message.MessageFrom(), rootHashMeta.RootHashes)
	d.checkMajorMeta()
}

func (d *Download) setPIDRootHashesMap(pid string, rootHashesByte [][]byte) {
	rootHashes := make([]common.Hash, len(rootHashesByte))
	for i, rootHash := range rootHashesByte {
		rootHashes[i] = common.BytesToHash(rootHash)
	}
	d.pidRootHashesMap[pid] = rootHashes
}

func (d *Download) setRootHashPIDsMap(pid string, rootHashesByte [][]byte) {
	for _, rootHash := range rootHashesByte {
		d.rootHashPIDsMap[common.BytesToHash(rootHash)][pid] = struct{}{}
	}
}

func (d *Download) checkMajorMeta() {
	if len(d.pidRootHashesMap) < d.majorityThreshold {
		return
	}
	//i := uint64(0)
	i := len(d.runningTasks) + len(d.finishedTasks) + len(d.taskQue)
	for {
		peerCounter := make(map[common.Hash][]string)
		for pid, rootHashes := range d.pidRootHashesMap {
			if len(rootHashes) < i {
				peerCounter[rootHashes[i]] = append(peerCounter[rootHashes[i]], pid)
			}
		}
		majorNotFound := true
		for rootHash, peers := range peerCounter {
			if len(peers) > d.majorityThreshold {
				//createDownloadTask
				majorNotFound = false
				t := NewDownloadTask(d.netService, d.rootHashPIDsMap[rootHash], d.from+uint64(i)*d.chunkSize, d.chunkSize, rootHash, d.taskDoneCh)
				d.taskAddCh <- t
			}
		}
		if majorNotFound {
			break
		}
		i++
	}
}

func (d *Download) findTaskForBlockChunk(message net.Message) {

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

	for _, t := range d.runningTasks {
		if t.from == t.from {
			t.blockChunkMessageCh <- message
		}
	}
}

func (d *Download) sendMetaQuery() error {
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
	return nil
}

func addTask(tasks []*DownloadTask, task *DownloadTask) {
	tasks = append(tasks, task)
}

func removeTask(tasks []*DownloadTask, task *DownloadTask) {
	for i, t := range tasks {
		if t == task {
			tasks = append(tasks[:i], tasks[i+1:]...)
		}
	}
}
