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
	ns        net.Service
	consensus Consensus

	receiveBlockMessageCh chan net.Message
	requestBlockMessageCh chan net.Message
	quitCh                chan int
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
		bc: bc,
		bp: bp,
		receiveBlockMessageCh: make(chan net.Message, defaultBlockMessageChanSize),
		requestBlockMessageCh: make(chan net.Message, defaultBlockMessageChanSize),
		quitCh:                make(chan int, 1),
	}, nil
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
	go bm.loop()
}

// Stop stops BlockManager service.
func (bm *BlockManager) Stop() {
	logging.Console().Info("Stopping BlockManager...")
	bm.quitCh <- 0
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
func (bm *BlockManager) BlockByHeight(height uint64) *Block {
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

// Relay relays BlockData to network.
func (bm *BlockManager) Relay(bd *BlockData) {
	bm.ns.Relay(MessageTypeNewBlock, bd, net.MessagePriorityHigh)
}

// BroadCast broadcasts BlockData to network.
func (bm *BlockManager) BroadCast(bd *BlockData) {
	bm.ns.Broadcast(MessageTypeNewBlock, bd, net.MessagePriorityHigh)
}

// PushBlockData pushes block data.
func (bm *BlockManager) PushBlockData(bd *BlockData) error {
	return bm.push(bd)
}

func (bm *BlockManager) push(bd *BlockData) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()
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

	// TODO @cl9200 Filter blocks of same height.

	if err := bd.VerifyIntegrity(); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to verify block signatures.")
		return err
	}

	if err := bm.consensus.VerifyProposer(bm.bc, bd); err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Debug("Failed to verify blockData.")
		return err
	}

	if err := bm.bp.Push(bd); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Error("Failed to push to block pool.")
		return err
	}

	// Parent block doesn't exist in blockchain.
	parentOnChain := bm.bc.BlockByHash(bd.ParentHash())
	if parentOnChain == nil {
		return nil
	}

	if bm.bc.IsForkedBeforeLIB(parentOnChain) {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Received a block forked before current LIB.")
		return ErrCannotRevertLIB
	}

	// Parent block exists in blockchain.
	all, tails, fails := bm.findDescendantBlocks(parentOnChain)
	for _, fail := range fails {
		bm.bp.Remove(fail)
	}
	if len(all) == 0 {
		logging.Console().WithFields(logrus.Fields{
			"parent": parentOnChain,
			"block":  bd,
		}).Error("Failed to find descendant blocks.")
		return ErrCannotExecuteOnParentBlock
	}

	bm.bc.PutVerifiedNewBlocks(parentOnChain, all, tails)

	for _, block := range all {
		bm.bp.Remove(block)
	}

	newTail := bm.consensus.ForkChoice(bm.bc)
	err := bm.bc.SetTailBlock(newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set new tail block.")
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
		"block": bd,
		"tail":  newTail,
		"lib":   newLIB,
	}).Info("Block pushed.")

	return nil
}

func (bm *BlockManager) findDescendantBlocks(parent *Block) (all []*Block, tails []*Block, fails []*BlockData) {
	children := bm.bp.FindChildren(parent)
	for _, v := range children {
		childData := v.(*BlockData)
		block, err := childData.ExecuteOnParentBlock(parent)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":    err,
				"block":  block,
				"parent": parent,
			}).Warn("Failed to execute on a parent block.")
			fails = append(fails, childData)
			continue
		}

		childAll, childTails, childFail := bm.findDescendantBlocks(block)

		all = append(all, block)
		all = append(all, childAll...)

		if len(childTails) == 0 {
			tails = append(tails, block)
		} else {
			tails = append(tails, childTails...)
		}

		fails = append(fails, childFail...)
	}
	return all, tails, fails
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

	downloadMsg := &corepb.DownloadParentBlock{
		Hash: unlinkedBlock.Hash(),
		Sign: unlinkedBlock.Signature(),
	}
	bytes, err := proto.Marshal(downloadMsg)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": unlinkedBlock,
		}).Debug("Failed to marshal download parent request.")
		return err
	}

	bm.ns.SendMsg(MessageTypeRequestBlock, bytes, sender, net.MessagePriorityNormal)
	return nil
}

func (bm *BlockManager) loop() {
	for {
		select {
		case <-bm.quitCh:
			return
		case msg := <-bm.receiveBlockMessageCh:
			go bm.handleReceiveBlock(msg)
		case msg := <-bm.requestBlockMessageCh:
			go bm.handleRequestBlock(msg)
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

	bd, err := bytesToBlockData(msg.Data())
	if err != nil {
		return
	}

	if msg.MessageType() == MessageTypeNewBlock {
		ts := time.Unix(bd.Timestamp(), 0)
		now := time.Now()
		if ts.After(now) && ts.Sub(now) > newBlockBroadcastTimeLimit || now.After(ts) && now.Sub(ts) > newBlockBroadcastTimeLimit {
			logging.WithFields(logrus.Fields{
				"block": bd,
				"now":   now,
			}).Warn("New block broadcast timeout.")
			return
		}
	}

	err = bm.push(bd)
	if err != nil {
		return
	}

	err = bm.requestMissingBlock(msg.MessageFrom(), bd)
	if err != nil {
		return
	}

	if msg.MessageType() == MessageTypeNewBlock {
		bm.Relay(bd)
	}
}

func (bm *BlockManager) handleRequestBlock(msg net.Message) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	if msg.MessageType() != MessageTypeRequestBlock {
		logging.WithFields(logrus.Fields{
			"msgType": msg.MessageType(),
			"msg":     msg,
		}).Debug("Received unregistered message.")
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

	if !byteutils.Equal(block.Signature(), pbDownloadParentBlock.Sign) {
		logging.WithFields(logrus.Fields{
			"download.hash": byteutils.Bytes2Hex(pbDownloadParentBlock.Hash),
			"download.sign": byteutils.Bytes2Hex(pbDownloadParentBlock.Sign),
			"expect.sign":   byteutils.Bytes2Hex(block.Signature()),
		}).Debug("Failed to check the block's signature.")
		return
	}

	parent := bm.bc.BlockByHash(block.ParentHash())
	if parent == nil {
		logging.WithFields(logrus.Fields{
			"block": block,
		}).Debug("Failed to find the block's parent.")
		return
	}

	bytes, err := net.SerializableToBytes(parent)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"parent": parent,
			"err":    err,
		}).Debug("Failed to serialize block's parent.")
		return
	}

	bm.ns.SendMsg(MessageTypeResponseBlock, bytes, msg.MessageFrom(), net.MessagePriorityNormal)

	logging.WithFields(logrus.Fields{
		"block":  block,
		"parent": parent,
	}).Debug("Responded to the download request.")
}
