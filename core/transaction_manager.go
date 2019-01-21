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
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	corestate "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/event"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Parameters for transaction manager
var (
	AllowReplacePendingDuration       = 10 * time.Minute
	defaultTransactionMessageChanSize = 128
)

// TransactionManager manages transactions' pool and network service.
type TransactionManager struct {
	mu sync.RWMutex

	chainID uint32

	receivedMessageCh chan net.Message
	quitCh            chan int
	canon             Canonical
	ns                net.Service

	pendingPool *PendingTransactionPool
	futurePool  *FutureTransactionPool
}

// NewTransactionManager create a new TransactionManager.
func NewTransactionManager(cfg *medletpb.Config) *TransactionManager {
	return &TransactionManager{
		chainID:           cfg.Global.ChainId,
		receivedMessageCh: make(chan net.Message, defaultTransactionMessageChanSize),
		quitCh:            make(chan int),
		pendingPool:       NewPendingTransactionPool(),
		futurePool:        NewFutureTransactionPool(int(cfg.Chain.TransactionPoolSize)),
	}
}

// Setup sets up TransactionManager.
func (tm *TransactionManager) Setup(canon Canonical, ns net.Service) {
	tm.canon = canon
	if ns != nil {
		tm.ns = ns
		tm.registerInNetwork()
	}
}

// InjectEmitter inject emitter generated from medlet to transaction manager
func (tm *TransactionManager) InjectEmitter(emitter *event.Emitter) {
	// TODO Event Emitter
	// tm.pendingPool.SetEventEmitter(emitter)
}

// Start starts TransactionManager.
func (tm *TransactionManager) Start() {
	logging.Console().WithFields(logrus.Fields{
		"MaxPendingByAccount": MaxPendingByAccount,
		"FuturePoolSize":      tm.futurePool.size,
	}).Info("Starting TransactionManager...")

	go tm.loop()
}

// Stop stops TransactionManager.
func (tm *TransactionManager) Stop() {
	tm.quitCh <- 1
}

// registerInNetwork register message subscriber in network.
func (tm *TransactionManager) registerInNetwork() {
	tm.ns.Register(net.NewSubscriber(tm, tm.receivedMessageCh, true, MessageTypeNewTx, net.MessageWeightNewTx))
}

// Push pushes a transaction.
func (tm *TransactionManager) Push(tx *corestate.Transaction) error {
	txc, err := NewTxContext(tx)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tx":  tx,
			"err": err,
		}).Debug("Failed to new transaction context.")
		return err
	}
	return tm.push(txc)
}

// PushAndBroadcast pushes a transaction and broadcast when transaction transits to pending pool.
func (tm *TransactionManager) PushAndBroadcast(tx *corestate.Transaction) error {
	txc, err := NewTxContextWithBroadcast(tx)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tx":  tx,
			"err": err,
		}).Debug("Failed to new transaction context.")
		return err
	}
	return tm.push(txc)
}

func (tm *TransactionManager) push(txc *TxContext) error {
	err := txc.VerifyIntegrity(tm.chainID)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed to verify transaction.")
		return err
	}

	bs := tm.canon.TailBlock().State()
	price := bs.Price()
	from := txc.From()
	acc, err := bs.GetAccount(from)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get account.")
		return err
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()
	nonceUpperLimit := tm.pendingPool.NonceUpperLimit(acc)
	if txc.Nonce() > nonceUpperLimit {
		// TODO emit event
		evicted := tm.futurePool.Set(txc)
		if byteutils.Equal(evicted.Hash(), txc.Hash()) {
			// TODO Err Type
			return errors.New("transaction pool is full")
		}
		return nil
	}

	err = tm.pendingPool.PushOrReplace(txc, acc, bs.Price())
	if err != nil {
		return err
	}
	if txc.broadcast {
		tm.Broadcast(txc.Transaction)
	}

	// TODO Refactor extract method(transitTx)
	for {
		nonceUpperLimit = tm.pendingPool.NonceUpperLimit(acc)
		txc = tm.futurePool.PopWithNonceUpperLimit(from, nonceUpperLimit)
		if txc == nil {
			break
		}
		err = tm.pendingPool.PushOrReplace(txc, acc, price)
		if err != nil {
			break
		}
		if txc.broadcast {
			tm.Broadcast(txc.Transaction)
		}
	}

	return nil
}

// ResetTransactionSelector resets transaction selector.
func (tm *TransactionManager) ResetTransactionSelector() {
	tm.pendingPool.ResetSelector()
}

// Next returns next transaction from transaction selector.
func (tm *TransactionManager) Next() *corestate.Transaction {
	tx := tm.pendingPool.Next()
	if tx != nil {
		return nil
	}
	return tx
}

// SetRequiredNonce sets required nonce for given address to transaction selector.
func (tm *TransactionManager) SetRequiredNonce(addr common.Address, nonce uint64) {
	tm.pendingPool.SetRequiredNonce(addr, nonce)
}

// Get transaction from transaction pool.
func (tm *TransactionManager) Get(hash []byte) *corestate.Transaction {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tx := tm.pendingPool.Get(hash)
	if tx != nil {
		return tx
	}
	txc := tm.futurePool.Get(hash)
	if txc == nil {

	}
	return txc.Transaction
}

// DelByAddressNonce del transaction specific addr-nonce transaction
// TODO Refactoring (addr, nonce) => Account
func (tm *TransactionManager) DelByAddressNonce(addr common.Address, nonce uint64) error {
	tm.mu.Lock()
	tm.pendingPool.Prune(addr, nonce)
	tm.futurePool.Prune(addr, nonce, nil)
	tm.mu.Unlock()

	bs := tm.canon.TailBlock().State()
	price := bs.Price()
	from := addr
	acc, err := bs.GetAccount(addr)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get account.")
		return err
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()
	for {
		nonceUpperLimit := tm.pendingPool.NonceUpperLimit(acc)
		txc := tm.futurePool.PopWithNonceUpperLimit(from, nonceUpperLimit)
		if txc == nil {
			break
		}
		err = tm.pendingPool.PushOrReplace(txc, acc, price)
		if err != nil {
			break
		}
		if txc.broadcast {
			tm.Broadcast(txc.Transaction)
		}
	}
	return nil
}

// TODO Need GETALL(?)
// // GetAll returns all transactions from transaction pool
// func (tm *TransactionManager) GetAll() []*Transaction {
// 	pending := tm.pendingPool.GetAll()
// 	future := tm.futurePool.GetAll()
// 	return append(pending, future...)
// }

// Broadcast broadcasts transaction to network.
func (tm *TransactionManager) Broadcast(tx *corestate.Transaction) error {
	b, err := tx.ToBytes()
	if err != nil {
		return err
	}

	tm.ns.Broadcast(MessageTypeNewTx, b, net.MessagePriorityNormal)
	return nil
}

func (tm *TransactionManager) loop() {
	for {
		select {
		case <-tm.quitCh:
			logging.Console().Info("Stopped TransactionManager...")
			return
		case msg := <-tm.receivedMessageCh:
			tx, err := txFromNetMsg(msg)
			if err != nil {
				continue
			}
			tm.PushAndBroadcast(tx)
		}
	}
}

func txFromNetMsg(msg net.Message) (*corestate.Transaction, error) {
	if msg.MessageType() != MessageTypeNewTx {
		logging.WithFields(logrus.Fields{
			"type": msg.MessageType(),
			"msg":  msg,
		}).Debug("Received unregistered message.")
		return nil, errors.New("invalid message type")
	}

	tx := new(corestate.Transaction)
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
