// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
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
	blockPoolSize               = 128
)

// BlockManager handles all logic related to BlockChain and BlockPool.
type BlockManager struct {
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
	bp, err := NewBlockPool(blockPoolSize)
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

	err := bm.bc.Setup(genesis, stor)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to setup blockchain.")
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
	return bm.bc.ChainID()
}

// BlockByHeight returns the block contained in the chain by height.
func (bm *BlockManager) BlockByHeight(height uint64) *Block {
	return bm.bc.BlockOnCanonicalByHeight(height)
}

// BlockByHash returns the block contained in the chain by hash.
func (bm *BlockManager) BlockByHash(hash common.Hash) *Block {
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
	if bm.bp.Has(bd) || bm.bc.BlockByHash(bd.Hash()) != nil {
		logging.WithFields(logrus.Fields{
			"blockData": bd,
		}).Debug("Found duplicated blockData.")
		return ErrDuplicatedBlock
	}

	// TODO @cl9200 Verify signature

	err := bm.consensus.VerifyProposer(bd)
	fmt.Println(bd, err)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Debug("Failed to verify blockData.")
		return err
	}

	// Parent block exists in blockpool.
	if bm.bp.FindParent(bd) != nil {
		bm.bp.Push(bd)
		return nil
	}

	// Parent block doesn't exist in blockchain.
	parentOnChain := bm.bc.BlockByHash(bd.ParentHash())
	if parentOnChain == nil {
		bm.bp.Push(bd)
		return nil
	}

	// Parent block exists in blockchain.
	err = bm.bp.Push(bd)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to push BlockData to BlockPool.")
		return err
	}
	all, tails, err := bm.findDescendantBlocks(parentOnChain)

	bm.bc.PutVerifiedNewBlocks(parentOnChain, all, tails)

	for _, block := range all {
		bm.bp.Remove(block)
	}

	newTail := bm.consensus.ForkChoice(bm.bc)
	err = bm.bc.SetTailBlock(newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set new tail block.")
		return err
	}

	return nil
}

func (bm *BlockManager) findDescendantBlocks(parent *Block) (all []*Block, tails []*Block, err error) {
	children := bm.bp.FindChildren(parent)
	for _, v := range children {
		childData := v.(*BlockData)
		block, err := childData.ExecuteOnParentBlock(parent)
		if err != nil {
			return nil, nil, err
		}

		childAll, childTails, err := bm.findDescendantBlocks(block)
		if err != nil {
			return nil, nil, err
		}

		all = append(all, block)
		all = append(all, childAll...)

		if len(childTails) == 0 {
			tails = append(tails, block)
		} else {
			tails = append(tails, childTails...)
		}
	}
	return all, tails, nil
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

	v := bm.bp.FindUnlinkedAncestor(bd)
	unlinkedBlock := v.(*BlockData)

	downloadMsg := &corepb.DownloadParentBlock{
		Hash: unlinkedBlock.Hash().Bytes(),
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
		// TODO @cl9200 Concurrent message processing with goroutine.
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
	if msg.MessageType() != MessageTypeNewBlock || msg.MessageType() != MessageTypeResponseBlock {
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

	// TODO @cl9200 Timeout check if MessageTypeNewBlock

	err = bm.push(bd)
	if err != nil {
		return
	}

	err = bm.requestMissingBlock(msg.MessageFrom(), bd)
	if err != nil {
		return
	}

	// TODO @cl9200 Should we relay if it is MessageTypeResponseBlock?
	bm.Relay(bd)
}

func (bm *BlockManager) handleRequestBlock(msg net.Message) {
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

	if byteutils.Equal(pbDownloadParentBlock.Hash, GenesisHash.Bytes()) {
		logging.WithFields(logrus.Fields{
			"hash": byteutils.Bytes2Hex(pbDownloadParentBlock.Hash),
		}).Debug("Asked to download genesis's parent, ignore it.")
		return
	}

	block := bm.bc.BlockByHash(common.BytesToHash(pbDownloadParentBlock.Hash))
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
