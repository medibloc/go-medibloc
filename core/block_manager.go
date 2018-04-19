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
	"github.com/medibloc/go-medibloc/util/logging"
)

// BlockManager handle all logics related to BlockChain & BlockPool
type BlockManager struct {
	bc *BlockChain
	bp *BlockPool
}

// ChainID return BlockChain.ChainID
func (bm *BlockManager) ChainID() uint32 {
	return bm.bc.ChainID()
}

type onParentNotExist func(common.Hash)

// HandleReceivedBlock handle received BlockData
func (bm *BlockManager) HandleReceivedBlock(block *BlockData, cb onParentNotExist) error {
	// TODO check if block already exist in storage
	err := block.VerifyIntegrity()
	if err != nil {
		return err
	}
	parentHash := block.ParentHash()
	parentBlock := bm.bc.GetBlock(parentHash)
	if parentBlock == nil {
		bm.bp.Push(block)
		if cb != nil {
			cb(parentHash)
		}
		return nil
	}
	// TODO allBlocks => better name
	allBlocks, tailBlocks, err := blocksFromBlockPool(parentBlock, block, bm.bp)
	if err != nil {
		return err
	}

	err = bm.bc.PutVerifiedNewBlocks(parentBlock, allBlocks, tailBlocks)
	if err != nil {
		return err
	}

	if len(allBlocks) > 1 {
		for _, b := range allBlocks[1:] {
			bm.bp.Remove(b)
		}
	}

	return nil
}

// TailBlock getter for mainTailBlock
func (bm *BlockManager) TailBlock() *Block {
	return bm.bc.MainTailBlock()
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
