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

package core

import (
	"bytes"
	"time"

	"github.com/gogo/protobuf/proto"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

var (
	defaultBlockMessageChanSize = 128
	newBlockBroadcastTimeLimit  = 3 * time.Second
)

// BlockManager handles all logic related to BlockChain and BlockPool.
type BlockManager struct {
	bc        *BlockChain
	bp        *BlockPool
	tm        *TransactionManager
	ns        net.Service
	consensus Consensus
	txMap     TxFactory

	syncService          SyncService
	syncActivationHeight uint64

	receiveBlockMessageCh chan net.Message
	requestBlockMessageCh chan net.Message
	quitCh                chan int

	// If worker finishes it's work and new block is accepted on the chain distributor put children blockdata on the
	// workQ
	finishWorkCh chan *blockResult
	// NewBlock from network
	newBlockCh     chan *blockPackage
	closeWorkersCh chan bool
	workFinishedCh chan bool
}

type blockPackage struct {
	*BlockData
	okCh   chan bool
	execCh chan bool
}

type blockResult struct {
	block   *blockPackage
	isValid bool
}

//TxMap returns txMap
func (bm *BlockManager) TxMap() TxFactory {
	return bm.txMap
}

//SetTxMap inject txMap
func (bm *BlockManager) SetTxMap(txMap TxFactory) {
	bm.txMap = txMap
}

// NewBlockManager returns BlockManager.
func NewBlockManager(cfg *medletpb.Config) (*BlockManager, error) {
	bp, err := NewBlockPool(int(cfg.Chain.BlockPoolSize))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create blockpool.")
		return nil, err
	}
	bc, err := NewBlockChain(cfg)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create blockchain.")
		return nil, err
	}
	return &BlockManager{
		bc:                    bc,
		bp:                    bp,
		syncActivationHeight:  cfg.Sync.SyncActivationHeight,
		receiveBlockMessageCh: make(chan net.Message, defaultBlockMessageChanSize),
		requestBlockMessageCh: make(chan net.Message, defaultBlockMessageChanSize),
		finishWorkCh:          make(chan *blockResult, 100), // TODO @ggomma use config
		newBlockCh:            make(chan *blockPackage, 100),
		quitCh:                make(chan int),
		closeWorkersCh:        make(chan bool),
		workFinishedCh:        make(chan bool),
	}, nil
}

// InjectEmitter inject emitter generated from medlet to block manager
func (bm *BlockManager) InjectEmitter(emitter *EventEmitter) {
	bm.bc.SetEventEmitter(emitter)
}

// InjectTransactionManager inject transaction manager from medlet to block manager
func (bm *BlockManager) InjectTransactionManager(tm *TransactionManager) {
	bm.tm = tm
}

//InjectSyncService inject sync service generated from medlet to block manager
func (bm *BlockManager) InjectSyncService(syncService SyncService) {
	bm.syncService = syncService
}

// Setup sets up BlockManager.
func (bm *BlockManager) Setup(genesis *corepb.Genesis, stor storage.Storage, ns net.Service, consensus Consensus, txMap TxFactory) error {
	bm.consensus = consensus
	bm.txMap = txMap

	err := bm.bc.Setup(genesis, consensus, txMap, stor)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to setup Blockchain.")
		return err
	}

	if ns != nil {
		bm.ns = ns
		bm.registerInNetwork()
	}
	return nil
}

// Start starts BlockManager service.
func (bm *BlockManager) Start() {
	logging.Console().Info("Starting BlockManager...")
	go bm.runDistributor()
	go bm.loop()
}

// Stop stops BlockManager service.
func (bm *BlockManager) Stop() {
	logging.Console().Info("Stopping BlockManager...")

	bm.closeWorkersCh <- true
	<-bm.workFinishedCh
	close(bm.quitCh)
}

func (bm *BlockManager) processTask(newData *blockPackage) {
	if b := bm.bc.BlockByHash(newData.Hash()); b != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": newData,
		}).Warn("Block is already on the chain")
		bm.alarmExecutionResult(newData, false)
		return
	}

	// Worker only handles the block data which parent is already on the chain
	parent := bm.bc.BlockByHash(newData.ParentHash())
	if parent == nil {
		logging.Console().WithFields(logrus.Fields{
			"block": newData,
		}).Warn("Failed to find parent block on the chain")
		bm.alarmExecutionResult(newData, false)
		return
	}

	if bm.bc.IsForkedBeforeLIB(parent) {
		logging.WithFields(logrus.Fields{
			"blockData": newData,
		}).Debug("Received a block forked before current LIB.")
		bm.alarmExecutionResult(newData, false)
		return
	}

	err := verifyBlockHeight(newData.BlockData, parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to verifyBlockHeight")
		bm.alarmExecutionResult(newData, false)
		return
	}

	err = verifyTimestamp(newData.BlockData, parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to verifyTimestamp")
		bm.alarmExecutionResult(newData, false)
		return
	}

	err = bm.consensus.VerifyInterval(newData.BlockData, parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"timestamp": newData.timestamp,
		}).Warn("Block timestamp is wrong")
		bm.alarmExecutionResult(newData, false)
		return
	}

	child, err := newData.ExecuteOnParentBlock(parent, bm.consensus, bm.txMap)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"parent": parent,
		}).Error("Failed to execute on a parent block.")
		bm.alarmExecutionResult(newData, false)
		return
	}

	if err := bm.bc.PutVerifiedNewBlock(parent, child); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"parent": parent,
			"child":  child,
		}).Error("Failed to put block on the chain.")
		bm.alarmExecutionResult(newData, false)
		return
	}

	newTail := bm.consensus.ForkChoice(bm.bc)

	revertBlocks, newBlocks, err := bm.bc.SetTailBlock(newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set new tail block.")
		bm.alarmExecutionResult(newData, false)
		return
	}
	if err := bm.rearrangeTransactions(revertBlocks, newBlocks); err != nil {
		bm.alarmExecutionResult(newData, false)
		return
	}

	newLIB := bm.consensus.FindLIB(bm.bc)
	err = bm.bc.SetLIB(newLIB)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set LIB.")
		bm.alarmExecutionResult(newData, false)
		return
	}
	logging.Console().WithFields(logrus.Fields{
		"block":       child,
		"ts":          time.Unix(newData.Timestamp(), 0),
		"tail_height": newTail.Height(),
		"lib_height":  newLIB.Height(),
	}).Info("Block pushed.")

	bm.alarmExecutionResult(newData, true)
	return
}

func (bm *BlockManager) runDistributor() {
	wm := newWorkQ()

	for {
		select {
		case result := <-bm.finishWorkCh:
			// remove from workQ
			wm.removeBlock(result.block.BlockData)
			bm.bp.Remove(result.block)

			// if block is invalid, remove block from pool
			if !result.isValid || wm.finish {
				break
			}

			children := bm.bp.FindChildren(result.block)
			for _, c := range children {
				wm.addBlock(c.(*blockPackage).BlockData)
				go bm.processTask(c.(*blockPackage))
			}
		case blockPackage := <-bm.newBlockCh:

			// skip if ancestor is already on workQ
			if wm.hasBlock(blockPackage.BlockData) {
				blockPackage.okCh <- false
				continue
			}

			// skip if ancestor's parent is not on the chain
			if bd := bm.bc.BlockByHash(blockPackage.ParentHash()); bd == nil {
				blockPackage.okCh <- false
				continue
			}

			wm.addBlock(blockPackage.BlockData)

			go bm.processTask(blockPackage)
			blockPackage.okCh <- true
		case <-bm.closeWorkersCh:
			wm.finishWork()
		}

		if wm.finish && len(wm.q) == 0 {
			bm.workFinishedCh <- true
			return
		}
	}
}

func (bm *BlockManager) registerInNetwork() {
	bm.ns.Register(net.NewSubscriber(bm, bm.receiveBlockMessageCh, true, MessageTypeNewBlock, net.MessageWeightNewBlock))
	bm.ns.Register(net.NewSubscriber(bm, bm.receiveBlockMessageCh, false, MessageTypeResponseBlock, net.MessageWeightZero))
	bm.ns.Register(net.NewSubscriber(bm, bm.requestBlockMessageCh, false, MessageTypeRequestBlock, net.MessageWeightZero))
}

// ChainID return BlockChain.ChainID
func (bm *BlockManager) ChainID() uint32 {
	return bm.bc.ChainID()
}

// BlockByHeight returns the block contained in the chain by height.
func (bm *BlockManager) BlockByHeight(height uint64) (*Block, error) {
	return bm.bc.BlockByHeight(height)
}

// BlockByHash returns the block contained in the chain by hash.
func (bm *BlockManager) BlockByHash(hash []byte) *Block {
	return bm.bc.BlockByHash(hash)
}

// TailBlock getter for mainTailBlock
func (bm *BlockManager) TailBlock() *Block {
	return bm.bc.MainTailBlock()
}

// LIB returns latest irreversible block of the chain.
func (bm *BlockManager) LIB() *Block {
	return bm.bc.LIB()
}

//ForceLIB set LIB force
func (bm *BlockManager) ForceLIB(b *Block) error {
	return bm.bc.SetLIB(b)
}

// Relay relays BlockData to network.
func (bm *BlockManager) Relay(bd *BlockData) {
	bm.ns.Relay(MessageTypeNewBlock, bd, net.MessagePriorityHigh)
}

// BroadCast broadcasts BlockData to network.
func (bm *BlockManager) BroadCast(bd *BlockData) {
	logging.Console().WithFields(logrus.Fields{
		"hash":   bd.Hash(),
		"height": bd.height,
	}).Info("block is broadcasted")
	bm.ns.Broadcast(MessageTypeNewBlock, bd, net.MessagePriorityHigh)
}

// PushBlockData pushes block data.
func (bm *BlockManager) PushBlockData(bd *BlockData) error {
	return bm.push(bd)
}

// PushCreatedBlock push block to block chain without execution and verification (only used for self made block)
func (bm *BlockManager) PushCreatedBlock(b *Block) error {
	return bm.directPush(b)
}

func (bm *BlockManager) directPush(b *Block) error {
	// Parent block doesn't exist in blockchain.
	parentOnChain := bm.bc.BlockByHash(b.ParentHash())
	if parentOnChain == nil {
		return ErrFailedToDirectPush
	}

	if err := bm.bc.PutVerifiedNewBlock(parentOnChain, b); err != nil {
		return ErrFailedToDirectPush
	}

	newTail := bm.consensus.ForkChoice(bm.bc)
	revertBlocks, newBlocks, err := bm.bc.SetTailBlock(newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set new tail block.")
		return err
	}
	if err := bm.rearrangeTransactions(revertBlocks, newBlocks); err != nil {
		return err
	}

	newLIB := bm.consensus.FindLIB(bm.bc)
	err = bm.bc.SetLIB(newLIB)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set LIB.")
	}

	logging.Console().WithFields(logrus.Fields{
		"block":       b,
		"ts":          time.Unix(b.Timestamp(), 0),
		"tail_height": newTail.Height(),
		"lib_height":  newLIB.Height(),
	}).Info("Block is directly pushed.")

	return nil
}

func (bm *BlockManager) push(bd *BlockData) error {
	if err := bm.verifyBlockData(bd); err != nil {
		return err
	}

	newBlockPackage := &blockPackage{
		BlockData: bd,
		okCh:      nil,
		execCh:    nil,
	}

	if err := bm.bp.Push(newBlockPackage); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Error("Failed to push to block pool.")
		return err
	}

	bm.pushToDistributor(newBlockPackage)

	return nil
}

func (bm *BlockManager) pushToDistributor(bp *blockPackage) {
	newBlockPackage := &blockPackage{
		bp.BlockData,
		make(chan bool),
		bp.execCh,
	}
	if v := bm.bp.FindUnlinkedAncestor(bp); v != nil {
		newBlockPackage.BlockData = v.(*blockPackage).BlockData
		newBlockPackage.execCh = v.(*blockPackage).execCh
	}

	bm.newBlockCh <- newBlockPackage
	<-newBlockPackage.okCh

	return
}

func (bm *BlockManager) verifyBlockData(bd *BlockData) error {
	if bm.bc.chainID != bd.ChainID() {
		return ErrInvalidChainID
	}

	if bm.bp.Has(bd) || bm.bc.BlockByHash(bd.Hash()) != nil {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Found duplicated blockData.")
		return ErrDuplicatedBlock
	}

	if bd.Height() <= bm.bc.LIB().Height() {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Received a block forked before current LIB.")
		return ErrCannotRevertLIB
	}

	if err := bd.VerifyIntegrity(); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to verify block signatures.")
		return err
	}
	return nil
}

func verifyTimestamp(bd *BlockData, parent *Block) error {
	if bd.Timestamp() <= parent.Timestamp() {
		return ErrInvalidTimestamp
	}
	return nil
}

func verifyBlockHeight(bd *BlockData, parent *Block) error {
	if parent.Height()+1 != bd.Height() {
		return ErrInvalidBlockHeight
	}
	return nil
}

func (bm *BlockManager) rearrangeTransactions(revertBlock []*Block, newBlocks []*Block) error {
	var txs = make(map[*Transaction]bool)

	for _, newBlock := range newBlocks {
		for _, tx := range newBlock.Transactions() {
			bm.tm.pool.Del(tx)
			txs[tx] = true
		}
	}

	// revert block
	for _, block := range revertBlock {
		for _, tx := range block.Transactions() {
			if _, ok := txs[tx]; !ok {
				// Return transactions
				err := bm.tm.Push(tx)
				if err != nil && err != ErrDuplicatedTransaction {
					return err
				}
			}
		}
		if bm.bc.eventEmitter != nil {
			event := &Event{
				Topic: TopicRevertBlock,
				Data:  byteutils.Bytes2Hex(block.Hash()),
				Type:  "",
			}
			bm.bc.eventEmitter.Trigger(event)
		}
		logging.Console().Warn("A block is reverted.")
	}
	return nil
}

func (bm *BlockManager) alarmExecutionResult(bp *blockPackage, success bool) {
	if bp.execCh != nil {
		bp.execCh <- success
	}

	result := &blockResult{
		bp,
		true,
	}

	if success {
		bm.finishWorkCh <- result
	} else {
		result.isValid = false
		bm.finishWorkCh <- result
	}
}

// requestMissingBlock requests a missing block to connect to blockchain.
func (bm *BlockManager) requestMissingBlock(sender string, bd *BlockData) error {
	// Block already in the chain.
	if bm.bc.BlockByHash(bd.Hash()) != nil {
		return nil
	}

	// Block not included in BlockPool.
	if !bm.bp.Has(bd) {
		return nil
	}

	unlinkedBlock := bd
	if v := bm.bp.FindUnlinkedAncestor(bd); v != nil {
		unlinkedBlock = v.(*blockPackage).BlockData
	}

	if b := bm.bc.BlockByHash(unlinkedBlock.ParentHash()); b != nil {
		return nil
	}

	downloadMsg := &corepb.DownloadParentBlock{
		Hash: unlinkedBlock.Hash(),
		Sign: unlinkedBlock.Sign(),
	}
	bytes, err := proto.Marshal(downloadMsg)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": unlinkedBlock,
		}).Debug("Failed to marshal download parent request.")
		return err
	}

	logging.Console().WithFields(logrus.Fields{
		"bm": bm,
	}).Info("request missing parent block")

	return bm.ns.SendMsg(MessageTypeRequestBlock, bytes, sender, net.MessagePriorityNormal)
}

func (bm *BlockManager) loop() {
	for {
		select {
		case <-bm.quitCh:
			return
		case msg := <-bm.receiveBlockMessageCh:
			bm.handleReceiveBlock(msg)
		case msg := <-bm.requestBlockMessageCh:
			bm.handleRequestBlock(msg)
		}
	}
}

func (bm *BlockManager) handleReceiveBlock(msg net.Message) {
	if msg.MessageType() != MessageTypeNewBlock && msg.MessageType() != MessageTypeResponseBlock {
		logging.WithFields(logrus.Fields{
			"msgType": msg.MessageType(),
			"msg":     msg,
		}).Debug("Received unregistered message.")
		return
	}

	bd, err := BytesToBlockData(msg.Data())
	if err != nil {
		return
	}

	if msg.MessageType() == MessageTypeNewBlock {
		ts := time.Unix(bd.Timestamp(), 0)
		now := time.Now()
		if (ts.After(now) && (ts.Sub(now) > newBlockBroadcastTimeLimit)) || (now.After(ts) && (now.Sub(ts) > newBlockBroadcastTimeLimit)) {
			logging.WithFields(logrus.Fields{
				"block": bd,
				"ts":    ts,
				"now":   now,
			}).Warn("New block broadcast timeout.")
			return
		}
	}

	err = bm.push(bd)
	if err != nil {
		return
	}
	ok := bm.activateSync(bd)
	if !ok {
		err = bm.requestMissingBlock(msg.MessageFrom(), bd)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"sender": msg.MessageFrom(),
				"block":  bd,
				"err":    err,
			}).Error("Failed to request missing block.")

			return
		}
	}

	if msg.MessageType() == MessageTypeNewBlock {
		bm.Relay(bd)
	}
}

func (bm *BlockManager) activateSync(bd *BlockData) bool {
	if bm.syncService == nil {
		return false
	}

	if bm.syncService.IsDownloadActivated() {
		logging.Console().Info("sync download is in progress")
		return true
	}

	if bd.Height() <= bm.bc.MainTailBlock().Height()+bm.syncActivationHeight {
		return false
	}

	if err := bm.syncService.ActiveDownload(bd.Height()); err != nil {
		logging.WithFields(logrus.Fields{
			"newBlockHeight":       bd.Height(),
			"mainTailBlockHeight":  bm.bc.MainTailBlock().Height(),
			"syncActivationHeight": bm.syncActivationHeight,
			"err":                  err,
		}).Debug("Failed to activate sync download manager.")
		return false
	}

	logging.Console().WithFields(logrus.Fields{
		"newBlockHeight":       bd.Height(),
		"mainTailBlockHeight":  bm.bc.MainTailBlock().Height(),
		"syncActivationHeight": bm.syncActivationHeight,
	}).Info("Sync download manager is activated.")
	return true
}

func (bm *BlockManager) handleRequestBlock(msg net.Message) {
	if msg.MessageType() != MessageTypeRequestBlock {
		logging.Console().WithFields(logrus.Fields{
			"msgType": msg.MessageType(),
			"msg":     msg,
		}).Error("Received unregistered message.")
		return
	}

	pbDownloadParentBlock := new(corepb.DownloadParentBlock)
	if err := proto.Unmarshal(msg.Data(), pbDownloadParentBlock); err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"msgType": msg.MessageType(),
			"msg":     msg,
		}).Debug("Failed to unmarshal download parent block msg.")
		return
	}

	if byteutils.Equal(pbDownloadParentBlock.Hash, GenesisHash) {
		logging.WithFields(logrus.Fields{
			"hash": byteutils.Bytes2Hex(pbDownloadParentBlock.Hash),
		}).Debug("Asked to download genesis's parent, ignore it.")
		return
	}

	block := bm.bc.BlockByHash(pbDownloadParentBlock.Hash)
	if block == nil {
		logging.WithFields(logrus.Fields{
			"hash": byteutils.Bytes2Hex(pbDownloadParentBlock.Hash),
		}).Debug("Failed to find the block asked for.")
		return
	}

	if !byteutils.Equal(block.Sign(), pbDownloadParentBlock.Sign) {
		logging.WithFields(logrus.Fields{
			"download.hash": byteutils.Bytes2Hex(pbDownloadParentBlock.Hash),
			"download.sign": byteutils.Bytes2Hex(pbDownloadParentBlock.Sign),
			"expect.sign":   byteutils.Bytes2Hex(block.Sign()),
		}).Debug("Failed to check the block's signature.")
		return
	}

	parent := bm.bc.BlockByHash(block.ParentHash())
	if parent == nil {
		logging.Console().WithFields(logrus.Fields{
			"block": block,
		}).Error("Failed to find the block's parent.")
		return
	}

	bytes, err := net.SerializableToBytes(parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"parent": parent,
			"err":    err,
		}).Error("Failed to serialize block's parent.")
		return
	}

	err = bm.ns.SendMsg(MessageTypeResponseBlock, bytes, msg.MessageFrom(), net.MessagePriorityNormal)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"receiver": msg.MessageFrom(),
			"err":      err,
		}).Error("Failed to send block response message..")
		return
	}

	logging.WithFields(logrus.Fields{
		"block":  block,
		"parent": parent,
	}).Debug("Responded to the download request.")
}

type workManager struct {
	q      []*BlockData
	finish bool
}

func newWorkQ() *workManager {
	return &workManager{
		q:      make([]*BlockData, 0),
		finish: false,
	}
}

func (wm *workManager) hasBlock(bd *BlockData) bool {
	for _, b := range wm.q {
		if bytes.Equal(b.Hash(), bd.Hash()) {
			return true
		}
	}
	return false
}

func (wm *workManager) removeBlock(bd *BlockData) {
	for i, v := range wm.q {
		if bytes.Equal(v.Hash(), bd.Hash()) {
			wm.q = append(wm.q[:i], wm.q[i+1:]...)
			return
		}
	}
}

func (wm *workManager) addBlock(bd *BlockData) {
	wm.q = append(wm.q, bd)
	return
}

func (wm *workManager) finishWork() {
	wm.finish = true
	return
}
