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

var bs *BlockSubscriber

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
func StartBlockSubscriber(netService net.Service, bp *BlockPool, bc *BlockChain) {
	bs = &BlockSubscriber{
		bc:         bc,
		bp:         bp,
		quitCh:     make(chan int),
		netService: netService,
	}
	newBlockCh := make(chan net.Message)
	requestBlockCh := make(chan net.Message)
	netService.Register(net.NewSubscriber(bs, newBlockCh, true, MessageTypeNewBlock, net.MessageWeightNewBlock))
	netService.Register(net.NewSubscriber(bs, requestBlockCh, false, MessageTypeRequestBlock, net.MessageWeightZero))
	go func() {
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
				err = bs.handleReceivedBlock(blockData, msg.MessageFrom())
				if err != nil {
					logging.Console().Error(err)
				}
			case msg := <-requestBlockCh:
				if err := handleRequestBlockMessage(msg, netService); err != nil {
					logging.Console().Error(err)
				}
			case <-bs.quitCh:
				logging.Console().Info("Stop Block Subscriber")
				return
			}
		}
	}()
}

//StopBlockSubscriber stops BlockSubscriber.
func StopBlockSubscriber() {
	if bs != nil {
		bs.quitCh <- 0
	}
}

// PushBlock Temporarily for Test
func PushBlock(blockData *BlockData, bc *BlockChain, bp *BlockPool) error {
	subscriber := &BlockSubscriber{
		bc: bc,
		bp: bp,
	}
	return subscriber.handleReceivedBlock(blockData, "")
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

func handleRequestBlockMessage(msg net.Message, netService net.Service) error {
	logging.Info("handle request block message")
	pb := &corepb.DownloadBlock{}
	err := proto.Unmarshal(msg.Data(), pb)
	if err != nil {
		return err
	}
	logging.Console().Info("Request Block Message arrived")
	block := bs.bc.GetBlock(common.BytesToHash(pb.Hash))
	if block == nil {
		return errors.New("requested block does not exist")
	}
	data, err := net.SerializableToBytes(block)
	if err != nil {
		return err
	}
	return netService.SendMsg(MessageTypeNewBlock, data, msg.MessageFrom(), net.MessagePriorityNormal)
}

func (subscriber *BlockSubscriber) handleReceivedBlock(block *BlockData, sender string) error {
	// TODO check if block already exist in storage
	err := block.VerifyIntegrity()
	if err != nil {
		return err
	}
	parentHash := block.ParentHash()
	parentBlock := subscriber.bc.GetBlock(parentHash)
	if parentBlock == nil {
		subscriber.bp.Push(block)
		if subscriber.netService != nil && sender != "" {
			logging.Console().Infof("try to request parent block to %s", sender)
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
			return subscriber.netService.SendMsg(MessageTypeRequestBlock, bytes, sender, net.MessagePriorityHigh)
		}
		return nil
	}
	// Find all blocks from this block
	// TODO allBlocks => better name
	allBlocks, tailBlocks, err := blocksFromBlockPool(parentBlock, block, subscriber.bp)
	if err != nil {
		return err
	}

	// Add to tail
	err = subscriber.bc.PutVerifiedNewBlocks(parentBlock, allBlocks, tailBlocks)
	if err != nil {
		return err
	}

	if len(allBlocks) > 1 {
		for _, b := range allBlocks[1:] {
			subscriber.bp.Remove(b)
		}
	}

	return nil
}
