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

	receivedMessageCh chan net.Message
	quitCh            chan int

	pool *TransactionPool
	ns   net.Service
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
func (mgr *TransactionManager) Setup(ns net.Service) {
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
func (mgr *TransactionManager) Push(tx *Transaction) error {
	if err := tx.VerifyIntegrity(mgr.chainID); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tx":  tx,
			"err": err,
		}).Debug("Failed to verify tx.")
		return err
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
