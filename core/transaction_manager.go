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
	"errors"
	"fmt"

	"github.com/medibloc/go-medibloc/util"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

var defaultTransactionMessageChanSize = 128

// TransactionManager manages transactions' pool and network service.
type TransactionManager struct {
	chainID uint32
	bm      *BlockManager

	receivedMessageCh chan net.Message
	quitCh            chan int

	pool         *TransactionPool
	gappedTxPool *GappedTxPool
	ns           net.Service
}

// NewTransactionManager create a new TransactionManager.
func NewTransactionManager(cfg *medletpb.Config) *TransactionManager {
	return &TransactionManager{
		chainID:           cfg.Global.ChainId,
		receivedMessageCh: make(chan net.Message, defaultTransactionMessageChanSize),
		quitCh:            make(chan int),
		pool:              NewTransactionPool(int(cfg.Chain.TransactionPoolSize)),
	}
}

// Setup sets up TransactionManager.
func (mgr *TransactionManager) Setup(ns net.Service, bm *BlockManager) {
	mgr.bm = bm
	if ns != nil {
		mgr.ns = ns
		mgr.registerInNetwork()
	}
}

// InjectEmitter inject emitter generated from medlet to transaction manager
func (mgr *TransactionManager) InjectEmitter(emitter *EventEmitter) {
	mgr.pool.SetEventEmitter(emitter)
}

// Start starts TransactionManager.
func (mgr *TransactionManager) Start() {
	logging.Console().WithFields(logrus.Fields{
		"size": mgr.pool.size,
	}).Info("Starting TransactionManager...")

	go mgr.loop()
}

// Stop stops TransactionManager.
func (mgr *TransactionManager) Stop() {
	mgr.quitCh <- 1
}

// registerInNetwork register message subscriber in network.
func (mgr *TransactionManager) registerInNetwork() {
	mgr.ns.Register(net.NewSubscriber(mgr, mgr.receivedMessageCh, true, MessageTypeNewTx, net.MessageWeightNewTx))
}

// Push pushes transaction to TransactionManager.
func (mgr *TransactionManager) Push(tx *Transaction) error { // TODO change name? @jiseob
	from := tx.From().Str()

	fmt.Println(mgr.bm.bc.mainTailBlock)
	//fmt.Println(mgr.bm.TailBlock())
	//fmt.Println(mgr.bm.TailBlock().State())
	//state := mgr.bm.TailBlock().State()
	state := mgr.bm.bc.mainTailBlock.State()
	acc, err := state.AccState().GetAccount(tx.From())
	if err != nil {
		return err
	}
	cpuUsage, netUsage := uint64(1000), uint64(2000) //TODO replace with tx.Bandwidth
	ai := mgr.pool.getAccountInfo(from)
	cpuSum := ai.cpuSum + cpuUsage
	netSum := ai.netSum + netUsage //TODO lock
	cpuCost, err := state.cpuRef.Mul(util.NewUint128FromUint(cpuSum))
	if err != nil {
		return err
	}
	netCost, err := state.cpuRef.Mul(util.NewUint128FromUint(netSum))
	if err != nil {
		return err
	}
	cost, err := cpuCost.Add(netCost)
	if err != nil {
		return err
	}
	avail, err := acc.Vesting.Sub(acc.Bandwidth)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to calculate available usage.")
		return err
	}
	if avail.Cmp(cost) < 0 {
		return ErrBandwidthNotEnough
	}
	if err := mgr.pool.Push(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tx":  tx,
			"err": err,
		}).Info("Failed to push tx.")
		return err
	}
	return nil
}

// PushGappedTxPool pushes gapped future transaction to gappedTxPool of TransactionManager.
func (mgr *TransactionManager) PushGappedTxPool(tx *Transaction) error {

	if err := mgr.gappedTxPool.Push(tx); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tx":  tx,
			"err": err,
		}).Info("Failed to push tx.")
		return err
	}
	return nil
}

// Pop pop transaction from TransactionManager.
func (mgr *TransactionManager) Pop() *Transaction {
	return mgr.pool.Pop()
}

// Get transaction from transaction pool.
func (mgr *TransactionManager) Get(hash []byte) *Transaction {
	return mgr.pool.Get(hash)
}

// GetAll returns all transactions from transaction pool
func (mgr *TransactionManager) GetAll() []*Transaction {
	return mgr.pool.GetAll()
}

// GetByAddress returns transactions related with the address from transaction pool
func (mgr *TransactionManager) GetByAddress(address common.Address) []*Transaction {
	return mgr.pool.GetByAddress(address)
}

// Relay relays transaction to network.
func (mgr *TransactionManager) Relay(tx *Transaction) {
	mgr.ns.Relay(MessageTypeNewTx, tx, net.MessagePriorityNormal)
}

// Broadcast broadcasts transaction to network.
func (mgr *TransactionManager) Broadcast(tx *Transaction) {
	mgr.ns.Broadcast(MessageTypeNewTx, tx, net.MessagePriorityNormal)
}

func (mgr *TransactionManager) loop() {
	for {
		select {
		case <-mgr.quitCh:
			logging.Console().Info("Stopped TransactionManager...")
			return
		case msg := <-mgr.receivedMessageCh:
			tx, err := txFromNetMsg(msg)
			if err != nil {
				continue
			}
			if err := mgr.PushAndRelay(tx); err != nil {
				continue
			}
		}
	}
}

//PushAndRelay push and relay transaction
func (mgr *TransactionManager) PushAndRelay(tx *Transaction) error {
	if err := mgr.Push(tx); err != nil {
		return err
	}
	mgr.Relay(tx)
	return nil
}

//DisposeTx disposes transaction
func (mgr *TransactionManager) DisposeTx(tx *Transaction) error {
	if err := tx.VerifyIntegrity(mgr.chainID); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tx":  tx,
			"err": err,
		}).Debug("Failed to verify tx.")
		return err
	}

	block := mgr.bm.TailBlock() // TODO lock check @jiseob
	err := block.state.checkNonce(tx)
	if err == ErrSmallTransactionNonce {
		return err
	}
	from := tx.From()
	v := mgr.pool.buckets.Get(from.String()) // TODO make method (bucket 호출 말것)
	poolMaxNonce := v.(*bucket).peekLast().nonce
	if tx.nonce <= poolMaxNonce {
		err = mgr.replaceTx(tx)
		if err != nil {
			return err
		}
		return nil
	}
	if tx.nonce == poolMaxNonce+1 {
		var txs []*Transaction
		txs = append(txs, tx)
		txs = append(txs, mgr.gappedTxPool.PopContinuousTxs(tx)...)
		for _, tx := range txs {
			err := mgr.PushAndBroadcast(tx)
			if err != nil {
				return err
			}
		}
		return nil
	}
	mgr.PushGappedTxPool(tx)
	return nil
}

//PushAndBroadcast push and broadcast transaction
func (mgr *TransactionManager) PushAndBroadcast(tx *Transaction) error {
	if err := mgr.Push(tx); err != nil {
		return err
	}
	mgr.Broadcast(tx)
	return nil
}

//replaceTx replaceTx from tx pool
func (mgr *TransactionManager) replaceTx(tx *Transaction) error {
	err := mgr.pool.deleteReplaceTxTarget(tx)
	if err != nil {
		return err
	}
	mgr.Push(tx)
	return nil
}

func txFromNetMsg(msg net.Message) (*Transaction, error) {
	if msg.MessageType() != MessageTypeNewTx {
		logging.WithFields(logrus.Fields{
			"type": msg.MessageType(),
			"msg":  msg,
		}).Debug("Received unregistered message.")
		return nil, errors.New("invalid message type")
	}

	tx := new(Transaction)
	pbTx := new(corepb.Transaction)
	if err := proto.Unmarshal(msg.Data(), pbTx); err != nil {
		logging.WithFields(logrus.Fields{
			"type": msg.MessageType(),
			"msg":  msg,
			"err":  err,
		}).Debug("Failed to unmarshal data.")
		return nil, errors.New("failed to unmarshal data")
	}

	if err := tx.FromProto(pbTx); err != nil {
		logging.WithFields(logrus.Fields{
			"type": msg.MessageType(),
			"msg":  msg,
			"err":  err,
		}).Debug("Failed to recover a tx from proto data.")
		return nil, errors.New("failed to recover from proto data")
	}
	return tx, nil
}
