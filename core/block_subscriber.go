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
	"github.com/medibloc/go-medibloc/util/logging"
)

var (
	MessageTypeNewBlock = "newblock"
)

func findChildren(block *Block, bp *BlockPool) ([]*Block, []*Block) {
	allBlocks := make([]*Block, 0)
	tailBlocks := make([]*Block, 0)
	blocks := make([]*Block, 0)
	blocks = append(blocks, block)
	for len(blocks) > 0 {
		curBlock := blocks[0]
		allBlocks = append(allBlocks)
		blocks = blocks[1:]
		children := bp.FindChildren(curBlock)
		if len(children) == 0 {
			tailBlocks = append(tailBlocks, curBlock)
		} else {
			blocks = append(blocks, children...)
		}
	}
	return allBlocks, tailBlocks
}

type BlockSubscriber struct {
	msgCh  chan net.Message
	quitCh chan int
}

var bs *BlockSubscriber

// StartBlockHandler
func StartBlockSubscriber(netService net.Service, bc *BlockChain, bp *BlockPool) {
	bs = &BlockSubscriber{
		msgCh:  make(chan net.Message),
		quitCh: make(chan int),
	}
	netService.Register(net.NewSubscriber(bs, bs.msgCh, true, MessageTypeNewBlock, net.MessageWeightNewBlock))
	for {
		select {
		case msg := <-bs.msgCh:
			block, err := bytesToBlock(msg.Data())
			if err != nil {
				logging.Error("failed to parse Block") // TODO
				continue
			}
			handleReceivedBlock(block, bc, bp)
		case <-bs.quitCh:
			return
		}
	}
}

// PushBlock Temporarily for Test
func PushBlock(block *Block, bc *BlockChain, bp *BlockPool) error {
	return handleReceivedBlock(block, bc, bp)
}

func handleReceivedBlock(block *Block, bc *BlockChain, bp *BlockPool) error {
	// TODO check if block already exist in storage
	err := block.VerifyIntegrity()
	if err != nil {
		return err
	}
	parentHash := block.ParentHash()
	parentBlock := bc.GetTailBlock(parentHash)
	if parentBlock == nil {
		// Add to BlockPool
		bp.Push(block)
		// TODO request parent block download
		return nil
	}
	// Find all blocks from this block
	allBlocks, tailBlocks := findChildren(block, bp)
	// TODO verify execution in parallel
	for _, b := range allBlocks {
		err = b.VerifyExecution()
		if err != nil {
			return err
		}
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
