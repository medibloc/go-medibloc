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
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
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
	mu        sync.RWMutex
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
	newBlockCh chan *blockPackage
	workQ      []*BlockData
}

type blockPackage struct {
	newBlock *BlockData
	okCh     chan bool
}

type blockResult struct {
	block      *BlockData
	blockError bool
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
		workQ:                 make([]*BlockData, 0, 100),
		quitCh:                make(chan int),
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
	//bm.quitCh <- 0
	close(bm.quitCh)
}

func (bm *BlockManager) runWorker(newData *BlockData) {
	bm.mu.Lock()

	result := &blockResult{
		newData,
		true,
	}

	if b := bm.bc.BlockByHash(newData.Hash()); b != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": newData,
		}).Warn("Block is already on the chain")
		bm.mu.Unlock()
		bm.finishWorkCh <- result
		return
	}

	// Worker only handles the block data which parent is already on the chain
	parent := bm.bc.BlockByHash(newData.ParentHash())
	if parent == nil {
		logging.Console().WithFields(logrus.Fields{
			"block": newData,
		}).Warn("Failed to find parent block on the chain")
		bm.mu.Unlock()
		bm.finishWorkCh <- result
		return
	}

	if bm.bc.IsForkedBeforeLIB(parent) {
		logging.WithFields(logrus.Fields{
			"blockData": newData,
		}).Debug("Received a block forked before current LIB.")
		bm.mu.Unlock()
		bm.finishWorkCh <- result
		return
	}
	bm.mu.Unlock()

	err := verifyBlockHeight(newData, parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to verifyBlockHeight")
		bm.finishWorkCh <- result
		return
	}

	err = verifyTimestamp(newData, parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to verifyTimestamp")
		bm.finishWorkCh <- result
		return
	}

	err = bm.consensus.VerifyInterval(newData, parent)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"timestamp": newData.timestamp,
		}).Warn("Block timestamp is wrong")
		bm.finishWorkCh <- result
		return
	}

	child, err := newData.ExecuteOnParentBlock(parent, bm.consensus, bm.txMap)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"parent": parent,
		}).Error("Failed to execute on a parent block.")
		bm.finishWorkCh <- result
		return
	}

	bm.mu.Lock()
	if err := bm.bc.PutVerifiedNewBlock(parent, child); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":    err,
			"parent": parent,
			"child":  child,
		}).Error("Failed to put block on the chain.")
		bm.mu.Unlock()
		bm.finishWorkCh <- result
		return
	}

	newTail := bm.consensus.ForkChoice(bm.bc)

	revertBlocks, newBlocks, err := bm.bc.SetTailBlock(newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set new tail block.")
		bm.mu.Unlock()
		bm.finishWorkCh <- result
		return
	}
	if err := bm.rearrangeTransactions(revertBlocks, newBlocks); err != nil {
		bm.mu.Unlock()
		bm.finishWorkCh <- result
		return
	}

	newLIB := bm.consensus.FindLIB(bm.bc)
	err = bm.bc.SetLIB(newLIB)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set LIB.")
		bm.mu.Unlock()
		bm.finishWorkCh <- result
		return
	}
	logging.Console().WithFields(logrus.Fields{
		"block":       child,
		"ts":          time.Unix(newData.Timestamp(), 0),
		"tail_height": newTail.Height(),
		"lib_height":  newLIB.Height(),
	}).Info("Block pushed.")

	// TODO @ggomma what if worker makes error?
	bm.mu.Unlock()
	result.blockError = true
	bm.finishWorkCh <- result
	return
}

func (bm *BlockManager) runDistributor() {
	// SETUP WORKER

	for {
		select {
		case result := <-bm.finishWorkCh:
			// remove from workQ
			for i, blockData := range bm.workQ {
				if blockData == result.block {
					bm.workQ = append(bm.workQ[0:i], bm.workQ[i+1:]...)
				}
			}

			bm.mu.Lock()
			bm.bp.Remove(result.block)
			bm.mu.Unlock()

			// if block is invalid, remove block from pool
			if result.blockError {
				continue
			}

			bm.mu.Lock()
			children := bm.bp.FindChildren(result.block)
			bm.mu.Unlock()

			for _, c := range children {
				bm.workQ[len(bm.workQ)] = c.(*BlockData)
				go bm.runWorker(c.(*BlockData))
			}
		case blockPackage := <-bm.newBlockCh:
			bm.mu.Lock()

			// skip if ancestor is already on workQ
			if err := bm.checkBlockInWorkQ(blockPackage.newBlock); err != nil {
				continue
			}

			// skip if ancestor's parent is not on the chain
			if bd := bm.bc.BlockByHash(blockPackage.newBlock.ParentHash()); bd == nil {
				continue
			}

			bm.workQ = append(bm.workQ, blockPackage.newBlock)
			bm.mu.Unlock()

			go bm.runWorker(blockPackage.newBlock)
			blockPackage.okCh <- true
		case <-bm.quitCh:
			return
		}
	}
}

func (bm *BlockManager) checkBlockInWorkQ(bd *BlockData) error {
	for _, b := range bm.workQ {
		if b == bd {
			return ErrAlreadyOnTheWorkQ
		}
	}
	return nil
}

func (bm *BlockManager) registerInNetwork() {
	bm.ns.Register(net.NewSubscriber(bm, bm.receiveBlockMessageCh, true, MessageTypeNewBlock, net.MessageWeightNewBlock))
	bm.ns.Register(net.NewSubscriber(bm, bm.receiveBlockMessageCh, false, MessageTypeResponseBlock, net.MessageWeightZero))
	bm.ns.Register(net.NewSubscriber(bm, bm.requestBlockMessageCh, false, MessageTypeRequestBlock, net.MessageWeightZero))
}

// ChainID return BlockChain.ChainID
func (bm *BlockManager) ChainID() uint32 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.bc.ChainID()
}

// BlockByHeight returns the block contained in the chain by height.
func (bm *BlockManager) BlockByHeight(height uint64) (*Block, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.bc.BlockByHeight(height)
}

// BlockByHash returns the block contained in the chain by hash.
func (bm *BlockManager) BlockByHash(hash []byte) *Block {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.bc.BlockByHash(hash)
}

// TailBlock getter for mainTailBlock
func (bm *BlockManager) TailBlock() *Block {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.bc.MainTailBlock()
}

// LIB returns latest irreversible block of the chain.
func (bm *BlockManager) LIB() *Block {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.bc.LIB()
}

//ForceLIB set LIB force
func (bm *BlockManager) ForceLIB(b *Block) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()
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
	bm.mu.Lock()
	defer bm.mu.Unlock()

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
	bm.mu.Lock()

	if bm.bc.chainID != bd.ChainID() {
		bm.mu.Unlock()
		return ErrInvalidChainID
	}

	if bm.bp.Has(bd) || bm.bc.BlockByHash(bd.Hash()) != nil {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Found duplicated blockData.")
		bm.mu.Unlock()
		return ErrDuplicatedBlock
	}

	if bd.Height() <= bm.bc.LIB().Height() {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Received a block forked before current LIB.")
		bm.mu.Unlock()
		return ErrCannotRevertLIB
	}

	// TODO @cl9200 Filter blocks of same height.
	if err := bd.VerifyIntegrity(); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to verify block signatures.")
		return err
	}

	if err := bm.bp.Push(bd); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Error("Failed to push to block pool.")
		return err
	}

	ancestor := bm.bp.FindUnlinkedAncestor(bd)

	blockPackage := &blockPackage{
		ancestor.(*BlockData),
		make(chan bool),
	}

	bm.mu.Unlock()
	bm.newBlockCh <- blockPackage
	<-blockPackage.okCh

	return nil
}

func (bm *BlockManager) findDescendantBlocks(parent *Block) (all []*Block, tails []*Block, fails []*BlockData) {
	children := bm.bp.FindChildren(parent)

	var wg sync.WaitGroup
	mu := &sync.Mutex{}

	for _, v := range children {
		wg.Add(1)
		go func(v HashableBlock) {
			defer wg.Done()
			childData := v.(*BlockData)

			err := verifyBlockHeight(childData, parent)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Warn("Failed to verifyBlockHeight")
				fails = append(fails, childData)
				return
			}

			err = verifyTimestamp(childData, parent)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Warn("Failed to verifyTimestamp")
				fails = append(fails, childData)
				return
			}

			err = bm.consensus.VerifyInterval(childData, parent)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err":       err,
					"timestamp": childData.timestamp,
				}).Warn("Block timestamp is wrong")
				fails = append(fails, childData)
				return
			}

			block, err := childData.ExecuteOnParentBlock(parent, bm.consensus, bm.txMap)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err":    err,
					"parent": parent,
				}).Warn("Failed to execute on a parent block.")
				fails = append(fails, childData)
				return
			}

			childAll, childTails, childFails := bm.findDescendantBlocks(block)

			mu.Lock()
			all = append(all, block)
			all = append(all, childAll...)

			if len(childTails) == 0 {
				tails = append(tails, block)
			} else {
				tails = append(tails, childTails...)
			}

			fails = append(fails, childFails...)
			mu.Unlock()
		}(v)
	}

	wg.Wait()
	return all, tails, fails
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
			}
			bm.bc.eventEmitter.Trigger(event)
		}
		logging.Console().Warn("A block is reverted.")
	}
	return nil
}

// requestMissingBlock requests a missing block to connect to blockchain.
func (bm *BlockManager) requestMissingBlock(sender string, bd *BlockData) error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	// Block already in the chain.
	if bm.bc.BlockByHash(bd.Hash()) != nil {
		return nil
	}

	// Block not included in BlockPool.
	if !bm.bp.Has(bd) {
		return nil
	}

	v := bm.bp.FindUnlinkedAncestor(bd)
	unlinkedBlock := v.(*BlockData)

	for _, bd := range bm.workQ {
		if bd == unlinkedBlock {
			return nil
		}
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
			"err": err,
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
	bm.mu.RLock()
	defer bm.mu.RUnlock()
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
