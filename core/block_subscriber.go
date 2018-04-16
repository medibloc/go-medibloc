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
	bm         *BlockManager
	netService net.Service
	quitCh     chan int
}

// NewBlockSubscriber create BlockSubscriber
func NewBlockSubscriber(bp *BlockPool, bc *BlockChain) *BlockSubscriber {
	return &BlockSubscriber{
		bm: &BlockManager{bc: bc, bp: bp},
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
				err = bs.handleReceivedBlock(blockData, msg.MessageFrom())
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

// GetBlockManager getter for BlockManager
func (bs *BlockSubscriber) GetBlockManager() *BlockManager {
	return bs.bm
}

// HandleReceivedBlock handle received BlockData
func (bs *BlockSubscriber) handleReceivedBlock(block *BlockData, sender string) error {
	return bs.bm.HandleReceivedBlock(block, func(parentHash common.Hash) {
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
			}
			err = bs.netService.SendMsg(MessageTypeRequestBlock, bytes, sender, net.MessagePriorityHigh)
			if err != nil {
				logging.Console().Error(err)
			}
		}
	})
}
