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
	"crypto/rand"
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Miner default configurations.
const (
	MinerInterval = 7 * time.Second
	MaxTxInBlock  = 100
)

// Miner creates blocks.
type Miner struct {
	quit chan int
}

// StartMiner starts miner.
func StartMiner(netService net.Service, bc *BlockChain, txMgr *TransactionManager) *Miner {
	miner := &Miner{quit: make(chan int)}
	go func() {
		ticker := time.NewTicker(MinerInterval)
		logging.Info("Start Miner")
		for {
			select {
			case <-ticker.C:
				logging.Console().Info("[Miner] Try to make block")
				err := makeBlock(netService, bc, txMgr)
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err": err,
					}).Error("Failed to make block")
				}
				logging.Console().Info("[Miner] New Block Created")
			case <-miner.quit:
				logging.Console().Info("Stop Miner")
				ticker.Stop()
				return
			}
		}
	}()
	return miner
}

// StopMiner stops miner.
func (miner *Miner) StopMiner() {
	miner.quit <- 0
}

func makeBlock(netService net.Service, bc *BlockChain, txMgr *TransactionManager) error {
	curTail := bc.MainTailBlock()
	// TODO get coinbase from config file
	var addr common.Address
	_, err := rand.Read(addr[:])
	if err != nil {
		return err
	}
	block, err := NewBlock(bc.ChainID(), addr, curTail)
	if err != nil {
		return err
	}

	txs := make(Transactions, 0)
	for len(txs) <= MaxTxInBlock {
		tx := txMgr.Pop()
		if tx == nil {
			break
		}
		txs = append(txs, tx)
	}
	block.SetTransactions(txs)
	err = block.ExecuteAll()
	if err != nil {
		return err
	}

	err = block.Seal()
	if err != nil {
		return err
	}
	// TODO block sign
	blocks := []*Block{block}
	err = bc.PutVerifiedNewBlocks(curTail, blocks, blocks)
	if err != nil {
		return err
	}
	netService.Broadcast(MessageTypeNewBlock, block, net.MessagePriorityHigh)
	return nil
}
