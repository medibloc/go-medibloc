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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/event"
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

var (
	finishWorkChannelSize = 128
	newBlockChannelSize   = 128
)

// BlockManager handles all logic related to BlockChain and BlockPool.
type BlockManager struct {
	bc        *BlockChain
	bp        *BlockPool
	tm        *TransactionManager
	ns        net.Service
	consensus Consensus

	syncService          SyncService
	syncActivationHeight uint64

	receiveBlockMessageCh chan net.Message
	requestBlockMessageCh chan net.Message
	quitCh                chan int

	// workManager
	finishWorkCh   chan *blockResult
	newBlockCh     chan *BlockData
	closeWorkersCh chan bool
	workFinishedCh chan bool

	// chainManager
	trigCh               chan bool
	finishChainManagerCh chan bool
	cmFinishedCh         chan bool
}

type blockResult struct {
	block   *BlockData
	isValid bool
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
		finishWorkCh:          make(chan *blockResult, finishWorkChannelSize),
		newBlockCh:            make(chan *BlockData, newBlockChannelSize),
		quitCh:                make(chan int),
		closeWorkersCh:        make(chan bool),
		workFinishedCh:        make(chan bool),
		trigCh:                make(chan bool, 1),
		finishChainManagerCh:  make(chan bool),
		cmFinishedCh:          make(chan bool),
	}, nil
}

// InjectEmitter inject emitter generated from medlet to block manager
func (bm *BlockManager) InjectEmitter(emitter *event.Emitter) {
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
func (bm *BlockManager) Setup(genesis *corepb.Genesis, stor storage.Storage, ns net.Service, consensus Consensus) error {
	bm.consensus = consensus

	err := bm.bc.Setup(genesis, consensus, stor)
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
	go bm.runChainManager()
	go bm.loop()
}

// Stop stops BlockManager service.
func (bm *BlockManager) Stop() {
	logging.Console().Info("Stopping BlockManager...")

	close(bm.closeWorkersCh)
	<-bm.workFinishedCh
	close(bm.finishChainManagerCh)
	<-bm.cmFinishedCh
	close(bm.quitCh)
}

func (bm *BlockManager) processTask(newData *BlockData) {
	if b := bm.bc.BlockByHash(newData.Hash()); b != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": newData,
		}).Warn("Block is already on the chain")
		bm.alarmExecutionResult(newData, ErrAlreadyOnTheChain)
		return
	}

	// Worker only handles the block data which parent is already on the chain
	parent := bm.bc.BlockByHash(newData.ParentHash())
	if parent == nil {
		logging.Console().WithFields(logrus.Fields{
			"block": newData,
		}).Warn("Failed to find parent block on the chain")
		bm.alarmExecutionResult(newData, ErrCannotFindParentBlockOnTheChain)
		return
	}

	if bm.bc.IsForkedBeforeLIB(parent) {
		logging.WithFields(logrus.Fields{
			"blockData": newData,
		}).Debug("Received a block forked before current LIB.")
		bm.alarmExecutionResult(newData, ErrForkedBeforeLIB)
		return
	}

	if err := verifyBlockHeight(newData, parent); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to verifyBlockHeight")
		bm.alarmExecutionResult(newData, err)
		return
	}

	if err := verifyTimestamp(newData, parent); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to verifyTimestamp")
		bm.alarmExecutionResult(newData, err)
		return
	}

	if err := bm.consensus.VerifyInterval(newData, parent); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"timestamp": newData.timestamp,
		}).Warn("Block timestamp is wrong")
		bm.alarmExecutionResult(newData, err)
		return
	}

	child, err := newData.ExecuteOnParentBlock(parent, bm.consensus)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"parent": parent,
		}).Error("Failed to execute on a parent block.")
		bm.alarmExecutionResult(newData, err)
		return
	}

	if err := bm.bc.PutVerifiedNewBlock(parent, child); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"parent": parent,
			"child":  child,
		}).Error("Failed to put block on the chain.")
		bm.alarmExecutionResult(newData, err)
		return
	}

	select {
	case bm.trigCh <- true:
	default:
	}

	logging.Console().WithFields(logrus.Fields{
		"block": child,
		"ts":    time.Unix(newData.Timestamp(), 0),
	}).Info("Block pushed.")
	bm.alarmExecutionResult(newData, nil)
	return
}

func (bm *BlockManager) runDistributor() {
	wm := newWorkManager()

	for {
		select {
		case result := <-bm.finishWorkCh:
			// remove from workQ
			wm.remove(result.block)
			bm.bp.Remove(result.block)

			// if block is invalid, remove block from pool
			if !result.isValid || wm.finish {
				break
			}

			children := bm.bp.FindChildren(result.block)
			for _, c := range children {
				wm.add(c.(*BlockData))
				go bm.processTask(c.(*BlockData))
			}
		case blockData := <-bm.newBlockCh:
			// skip if ancestor is already on workQ
			if wm.has(blockData) {
				continue
			}

			// skip if ancestor's parent is not on the chain
			if bd := bm.bc.BlockByHash(blockData.ParentHash()); bd == nil {
				continue
			}

			wm.add(blockData)
			go bm.processTask(blockData)
		case <-bm.closeWorkersCh:
			wm.finishWork()
		}

		if wm.finish && len(wm.blocks) == 0 {
			close(bm.workFinishedCh)
			return
		}
	}
}

func (bm *BlockManager) runChainManager() {
	mainTail := bm.TailBlock()
	LIB := bm.LIB()
	for {
		select {
		case <-bm.finishChainManagerCh:
			close(bm.cmFinishedCh)
			return
		case <-bm.trigCh:
			// newData is used only for alarming not affecting lib, tailblock, indexing process
			newTail := bm.consensus.ForkChoice(bm.bc)
			if byteutils.Equal(mainTail.Hash(), newTail.Hash()) {
				continue
			}
			mainTail = newTail

			revertBlocks, newBlocks, err := bm.bc.SetTailBlock(newTail)
			if err != nil {
				logging.WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to set new tail block.")
				continue
			}

			if err := bm.rearrangeTransactions(revertBlocks, newBlocks); err != nil {
				continue
			}

			newLIB := bm.consensus.FindLIB(bm.bc)
			if byteutils.Equal(LIB.Hash(), newLIB.Hash()) {
				continue
			}
			LIB = newLIB

			err = bm.bc.SetLIB(newLIB)
			if err != nil {
				logging.WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to set LIB.")
				continue
			}

			logging.Console().WithFields(logrus.Fields{
				"LIB":         newLIB,
				"newMainTail": newTail,
			}).Info("Block accepted.")
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

// BroadCast broadcasts BlockData to network.
func (bm *BlockManager) BroadCast(bd *BlockData) error {
	logging.Console().WithFields(logrus.Fields{
		"hash":   bd.Hash(),
		"height": bd.height,
	}).Info("block is broadcasted")

	b, err := bd.ToBytes()
	if err != nil {
		return err
	}
	bm.ns.Broadcast(MessageTypeNewBlock, b, net.MessagePriorityHigh)
	return nil
}

// PushBlockData pushes block data.
func (bm *BlockManager) PushBlockData(bd *BlockData) error {
	return bm.push(bd)
}

// PushCreatedBlock push block to block chain without execution and verification (only used for self made block)
func (bm *BlockManager) PushCreatedBlock(b *Block) error {
	return bm.directPush(b)
}

// PushBlockDataSync pushes block to distributor and wait for execution
// Warning! - Use this method only for test for time efficiency.
func (bm *BlockManager) PushBlockDataSync(bd *BlockData, timeLimit time.Duration) error {
	eventSubscriber, err := event.NewSubscriber(1024, []string{event.TopicAcceptedBlock, event.TopicInvalidBlock})
	if err != nil {
		return err
	}

	bm.bc.eventEmitter.Register(eventSubscriber)
	defer bm.bc.eventEmitter.Deregister(eventSubscriber)

	eCh := eventSubscriber.EventChan()

	if err := bm.push(bd); err != nil {
		return err
	}

	timeout := time.NewTimer(timeLimit)
	defer timeout.Stop()
	for {
		select {
		case e := <-eCh:
			if e.Data == byteutils.Bytes2Hex(bd.Hash()) {
				if e.Topic == event.TopicInvalidBlock {
					return ErrInvalidBlock
				}
				return nil
			}
		case <-timeout.C:
			return ErrBlockExecutionTimeout
		}
	}
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

	select {
	case bm.trigCh <- true:
	default:
	}

	logging.Console().WithFields(logrus.Fields{
		"block": b,
		"ts":    time.Unix(b.Timestamp(), 0),
	}).Info("Block is directly pushed.")

	return nil
}

func (bm *BlockManager) push(bd *BlockData) error {
	if err := bm.verifyBlockData(bd); err != nil {
		return err
	}

	if err := bm.bp.Push(bd); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Error("Failed to push to block pool.")
		return err
	}

	bm.pushToDistributor(bd)

	return nil
}

func (bm *BlockManager) pushToDistributor(bd *BlockData) {
	if ancestor := bm.bp.FindUnlinkedAncestor(bd); ancestor != nil {
		bm.newBlockCh <- ancestor.(*BlockData)
		return
	}
	bm.newBlockCh <- bd
	return
}

func (bm *BlockManager) verifyBlockData(bd *BlockData) error {
	if bm.bc.chainID != bd.ChainID() {
		return ErrInvalidBlockChainID
	}

	if bm.bp.Has(bd) || bm.bc.BlockByHash(bd.Hash()) != nil {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Found duplicated blockData.")
		return ErrDuplicatedBlock
	}

	if err := bm.consensus.VerifyHeightAndTimestamp(bm.bc.LIB().BlockData, bd); err != nil {
		//if bd.Height() <= bm.bc.LIB().Height() {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Received a block forked before current LIB.")
		return ErrFailedValidateHeightAndHeight
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
	addrNonce := make(map[common.Address]uint64)
	exclusiveFilter := make(map[*coreState.Transaction]bool)

	for _, newBlock := range newBlocks {
		for _, tx := range newBlock.Transactions() {
			if addrNonce[tx.From()] < tx.Nonce() {
				addrNonce[tx.From()] = tx.Nonce()
			}
			exclusiveFilter[tx] = true
		}
	}
	for addr, nonce := range addrNonce {
		bm.tm.DelByAddressNonce(addr, nonce)
	}

	// revert block
	pushTxs := make([]*coreState.Transaction, 0)
	for _, block := range revertBlock {
		for _, tx := range block.Transactions() {
			if _, ok := exclusiveFilter[tx]; !ok {
				pushTxs = append(pushTxs, tx)
			}
		}
		if bm.bc.eventEmitter != nil {
			block.EmitBlockEvent(bm.bc.eventEmitter, event.TopicRevertBlock)
		}
		logging.Console().Warn("A block is reverted.")
	}
	bm.tm.PushAndExclusiveBroadcast(pushTxs...)
	return nil
}

func (bm *BlockManager) alarmExecutionResult(bd *BlockData, error error) {
	result := &blockResult{
		bd,
		true,
	}

	if error == nil {
		bm.finishWorkCh <- result
	} else {
		result.isValid = false
		bm.finishWorkCh <- result
		bd.EmitBlockEvent(bm.bc.eventEmitter, event.TopicInvalidBlock)
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
		unlinkedBlock = v.(*BlockData)
	}

	if b := bm.bc.BlockByHash(unlinkedBlock.ParentHash()); b != nil {
		return nil
	}

	downloadMsg := &corepb.DownloadParentBlock{
		Hash: unlinkedBlock.Hash(),
		Sign: unlinkedBlock.Sign(),
	}
	byteMsg, err := proto.Marshal(downloadMsg)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": unlinkedBlock,
		}).Debug("Failed to marshal download parent request.")
		return err
	}

	logging.Console().WithFields(logrus.Fields{
		"blockHeight": bd.height,
		"to":          sender,
	}).Info("request missing parent block")

	bm.ns.SendMessageToPeer(MessageTypeRequestBlock, byteMsg, net.MessagePriorityNormal, sender)
	return nil
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
		if err := bm.BroadCast(bd); err != nil {
			logging.Console().WithFields(logrus.Fields{
				"sender": msg.MessageFrom(),
				"block":  bd,
				"err":    err,
			}).Error("Failed to broadcast block.")
			return
		}
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

	block := bm.bc.BlockByHash(pbDownloadParentBlock.Hash)
	if block == nil {
		logging.WithFields(logrus.Fields{
			"hash": byteutils.Bytes2Hex(pbDownloadParentBlock.Hash),
		}).Debug("Failed to find the block asked for.")
		return
	}

	if block.Height() == GenesisHeight {
		logging.WithFields(logrus.Fields{
			"hash":   byteutils.Bytes2Hex(pbDownloadParentBlock.Hash),
			"height": block.Height(),
		}).Debug("Asked to download genesis's parent, ignore it.")
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

	bytes, err := parent.ToBytes()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"parent": parent,
			"err":    err,
		}).Error("Failed to serialize block's parent.")
		return
	}

	bm.ns.SendMessageToPeer(MessageTypeResponseBlock, bytes, net.MessagePriorityNormal, msg.MessageFrom())

	logging.WithFields(logrus.Fields{
		"block":  block,
		"parent": parent,
	}).Debug("Responded to the download request.")
}

type workManager struct {
	blocks map[string]*BlockData
	finish bool
}

func newWorkManager() *workManager {
	return &workManager{
		blocks: make(map[string]*BlockData),
		finish: false,
	}
}

func (wm *workManager) has(bd *BlockData) bool {
	if wm.blocks[byteutils.Bytes2Hex(bd.Hash())] == nil {
		return false
	}
	return true
}

func (wm *workManager) remove(bd *BlockData) {
	delete(wm.blocks, byteutils.Bytes2Hex(bd.Hash()))
}

func (wm *workManager) add(bd *BlockData) {
	wm.blocks[byteutils.Bytes2Hex(bd.Hash())] = bd
}

func (wm *workManager) finishWork() {
	wm.finish = true
	return
}
