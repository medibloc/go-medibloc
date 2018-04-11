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
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

var (
	MessageTypeNewBlock = "newblock"
)

var (
	chainID                uint32 = 1010 // TODO
	defaultGenesisConfPath        = "conf/default/genesis.conf"
	blockPoolSize                 = 128
)

type BlockSubscriber struct {
	msgCh  chan net.Message
	quitCh chan int
}

var bs *BlockSubscriber

// GetBlockPoolBlockChain
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

// StartBlockHandler
func StartBlockSubscriber(netService net.Service, bp *BlockPool, bc *BlockChain) error {
	bs = &BlockSubscriber{
		msgCh:  make(chan net.Message),
		quitCh: make(chan int),
	}
	netService.Register(net.NewSubscriber(bs, bs.msgCh, true, MessageTypeNewBlock, net.MessageWeightNewBlock))
	go func() {
		for {
			select {
			case msg := <-bs.msgCh:
				logging.Console().Info("Block Message arrived")
				// TODO set storage to block
				// TODO make tries
				// TODO test
				block, err := bytesToBlockData(msg.Data())
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err": err,
					}).Error("failed to parse Block")
					continue
				}
				// logging.Console().Info("New Block Arrived")
				handleReceivedBlock(block, bc, bp)
			case <-bs.quitCh:
				logging.Console().Info("Stop Block Subscriber")
				return
			}
		}
	}()
	return nil
}

func StopBlockSubscriber() {
	if bs != nil {
		bs.quitCh <- 0
	}
}

// PushBlock Temporarily for Test
func PushBlock(blockData *BlockData, bc *BlockChain, bp *BlockPool) error {
	return handleReceivedBlock(blockData, bc, bp)
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

func handleReceivedBlock(block *BlockData, bc *BlockChain, bp *BlockPool) error {
	// TODO check if block already exist in storage
	err := block.VerifyIntegrity()
	if err != nil {
		return err
	}
	parentHash := block.ParentHash()
	parentBlock := bc.GetBlock(parentHash)
	if parentBlock == nil {
		// Add to BlockPool
		bp.Push(block)
		// TODO request parent block download
		return nil
	}
	// Find all blocks from this block
	// TODO allBlocks => better name
	allBlocks, tailBlocks, err := blocksFromBlockPool(parentBlock, block, bp)
	if err != nil {
		return err
	}

	// Add to tail
	err = bc.PutVerifiedNewBlocks(parentBlock, allBlocks, tailBlocks)
	if err != nil {
		return err
	}

	if len(allBlocks) > 1 {
		for _, b := range allBlocks[1:] {
			bp.Remove(b)
		}
	}

	return nil
}
