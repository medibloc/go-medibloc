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
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Block's message types.
const (
	MessageTypeNewBlock     = "newblock"
	MessageTypeRequestBlock = "reqblock"
)

var (
	chainID                uint32 = 1010 // TODO
	defaultGenesisConfPath        = "conf/default/genesis.conf"
	blockPoolSize                 = 128
)

// BlockSubscriber receives blocks from network.
type BlockSubscriber struct {
	bc         *BlockChain
	bp         *BlockPool
	netService net.Service
	quitCh     chan int
}

// NewBlockSubscriber create BlockSubscriber
func NewBlockSubscriber(bp *BlockPool, bc *BlockChain) *BlockSubscriber {
	return &BlockSubscriber{
		bc: bc,
		bp: bp,
	}
}

// GetBlockPoolBlockChain returns BlockPool and BlockChain.
func GetBlockPoolBlockChain(storage storage.Storage) (*BlockPool, *BlockChain, error) {
	bp, err := NewBlockPool(blockPoolSize)
	if err != nil {
		return nil, nil, err
	}
	conf, err := LoadGenesisConf(defaultGenesisConfPath)
	if err != nil {
		return nil, nil, err
	}
	genesisBlock, err := NewGenesisBlock(conf, storage)
	if err != nil {
		return nil, nil, err
	}
	bc, err := NewBlockChain(chainID, genesisBlock, storage)
	if err != nil {
		return nil, nil, err
	}
	return bp, bc, nil
}

// StartBlockSubscriber starts block subscribe service.
func StartBlockSubscriber(netService net.Service, bp *BlockPool, bc *BlockChain) *BlockSubscriber {
	bs := NewBlockSubscriber(bp, bc)
	bs.quitCh = make(chan int)
	bs.netService = netService
	go func() {
		newBlockCh := make(chan net.Message)
		requestBlockCh := make(chan net.Message)
		netService.Register(net.NewSubscriber(bs, newBlockCh, false, MessageTypeNewBlock, net.MessageWeightNewBlock))
		netService.Register(net.NewSubscriber(bs, requestBlockCh, false, MessageTypeRequestBlock, net.MessageWeightZero))
		for {
			select {
			case msg := <-newBlockCh:
				blockData, err := bytesToBlockData(msg.Data())
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err": err,
					}).Error("failed to parse blockData")
					continue
				}
				logging.Console().Info("Block arrived")
				err = bs.HandleReceivedBlock(blockData, msg.MessageFrom())
				if err != nil {
					logging.Console().Error(err)
				}
			case msg := <-requestBlockCh:
				if err := handleRequestBlockMessage(msg, bc, netService); err != nil {
					logging.Console().Error(err)
				}
			case <-bs.quitCh:
				logging.Console().Info("Stop Block Subscriber")
				return
			}
		}
	}()

	return bs
}

// StopBlockSubscriber stops BlockSubscriber.
func (bs *BlockSubscriber) StopBlockSubscriber() {
	if bs != nil {
		bs.quitCh <- 0
	}
}

func blocksFromBlockPool(parent *Block, blockData *BlockData, bp *BlockPool) ([]*Block, []*Block, error) {
	allBlocks := make([]*Block, 0)
	tailBlocks := make([]*Block, 0)
	queue := make([]*Block, 0)
	block, err := blockData.ExecuteOnParentBlock(parent)
	if err != nil {
		return nil, nil, err
	}
	queue = append(queue, block)
	for len(queue) > 0 {
		curBlock := queue[0]
		allBlocks = append(allBlocks, curBlock)
		queue = queue[1:]
		children1 := bp.FindChildren(curBlock)
		if len(children1) == 0 {
			tailBlocks = append(tailBlocks, curBlock)
		} else {
			children2 := make([]*Block, len(children1))
			for i, child := range children1 {
				children2[i], err = child.(*BlockData).ExecuteOnParentBlock(curBlock)
				if err != nil {
					return nil, nil, err
				}
			}
			queue = append(queue, children2...)
		}
	}
	return allBlocks, tailBlocks, nil
}

func handleRequestBlockMessage(msg net.Message, bc *BlockChain, netService net.Service) error {
	logging.Info("handle request block message")
	pb := &corepb.DownloadBlock{}
	err := proto.Unmarshal(msg.Data(), pb)
	if err != nil {
		return err
	}
	logging.Console().Info("Request Block Message arrived")
	block := bc.GetBlock(common.BytesToHash(pb.Hash))
	if block == nil {
		return errors.New("requested block does not exist")
	}
	data, err := net.SerializableToBytes(block)
	if err != nil {
		return err
	}
	return netService.SendMsg(MessageTypeNewBlock, data, msg.MessageFrom(), net.MessagePriorityNormal)
}

// HandleReceivedBlock handle received BlockData
func (bs *BlockSubscriber) HandleReceivedBlock(block *BlockData, sender string) error {
	// TODO check if block already exist in storage
	err := block.VerifyIntegrity()
	if err != nil {
		return err
	}
	parentHash := block.ParentHash()
	parentBlock := bs.bc.GetBlock(parentHash)
	if parentBlock == nil {
		bs.bp.Push(block)
		if bs.netService != nil && sender != "" {
			logging.Console().Info("try to request parent block")
			downloadMsg := &corepb.DownloadBlock{
				Hash: block.ParentHash().Bytes(),
			}
			bytes, err := proto.Marshal(downloadMsg)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"block": block,
					"err":   err,
				}).Error("failed to marshal download message")
				return err
			}
			return bs.netService.SendMsg(MessageTypeRequestBlock, bytes, sender, net.MessagePriorityHigh)
		}
		return nil
	}
	// TODO allBlocks => better name
	allBlocks, tailBlocks, err := blocksFromBlockPool(parentBlock, block, bs.bp)
	if err != nil {
		return err
	}

	err = bs.bc.PutVerifiedNewBlocks(parentBlock, allBlocks, tailBlocks)
	if err != nil {
		return err
	}

	if len(allBlocks) > 1 {
		for _, b := range allBlocks[1:] {
			bs.bp.Remove(b)
		}
	}

	return nil
}
