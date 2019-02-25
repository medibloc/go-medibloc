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
func (tm *TransactionManager) Push(tx *Transaction) error {
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
func (tm *TransactionManager) PushAndBroadcast(tx *Transaction) error {
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

	tail := tm.canon.TailBlock()
	enterPending, err := tm.pushToPool(tail, txc)
	if err != nil || !enterPending {
		return err
	}

	return tm.transit(tail, txc.From())
}

func (tm *TransactionManager) pushToPool(tail *Block, txc *TxContext) (enterPending bool, err error) {
	from, payer, err := getFromAndPayerAccount(tail.State(), txc)
	if err != nil {
		return false, err
	}
	price, err := tail.CalcChildPrice()
	if err != nil {
		return false, err
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()
	nonceUpperLimit := tm.pendingPool.NonceUpperLimit(from)
	if txc.Nonce() > nonceUpperLimit {
		// TODO emit event
		evicted := tm.futurePool.Set(txc)
		if evicted != nil && byteutils.Equal(evicted.Hash(), txc.Hash()) {
			// TODO Err Type
			return false, errors.New("transaction pool is full")
		}
		return false, nil
	}
	return true, tm.pushToPendingAndBroadcast(txc, from, payer, price)
}

func (tm *TransactionManager) transit(tail *Block, addr common.Address) error {
	for {
		tx := tm.futurePool.PeekLowerNonce(addr)
		if tx == nil {
			return nil
		}
		keepMoving, err := tm.moveBetweenPool(tail, tx)
		if err != nil || !keepMoving {
			return err
		}
	}
}

func (tm *TransactionManager) moveBetweenPool(tail *Block, txc *TxContext) (keepMoving bool, err error) {
	from, payer, err := getFromAndPayerAccount(tail.State(), txc)
	if err != nil {
		return false, err
	}
	price, err := tail.CalcChildPrice()
	if err != nil {
		return false, err
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()
	nonceUpperLimit := tm.pendingPool.NonceUpperLimit(from)
	if txc.Nonce() > nonceUpperLimit {
		return false, nil
	}
	if deleted := tm.futurePool.Del(txc); !deleted {
		return false, nil
	}
	err = tm.pushToPendingAndBroadcast(txc, from, payer, price)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (tm *TransactionManager) pushToPendingAndBroadcast(txc *TxContext, from *corestate.Account, payer *corestate.Account, price common.Price) error {
	err := tm.pendingPool.PushOrReplace(txc, from, payer, price)
	if err != nil {
		return err
	}
	if !txc.broadcast {
		return nil
	}
	return tm.Broadcast(txc.Transaction)
}

func getFromAndPayerAccount(bs *BlockState, txc *TxContext) (from, payer *corestate.Account, err error) {
	from, err = bs.GetAccount(txc.From())
	if err != nil {
		return nil, nil, err
	}
	if !txc.HasPayer() {
		return from, nil, nil
	}
	payer, err = bs.GetAccount(txc.Payer())
	if err != nil {
		return nil, nil, err
	}
	return from, payer, nil
}

// ResetTransactionSelector resets transaction selector.
func (tm *TransactionManager) ResetTransactionSelector() {
	tm.pendingPool.ResetSelector()
}

// Next returns next transaction from transaction selector.
func (tm *TransactionManager) Next() *Transaction {
	return tm.pendingPool.Next()
}

// SetRequiredNonce sets required nonce for given address to transaction selector.
func (tm *TransactionManager) SetRequiredNonce(addr common.Address, nonce uint64) {
	tm.pendingPool.SetRequiredNonce(addr, nonce)
}

// Get transaction from transaction pool.
func (tm *TransactionManager) Get(hash []byte) *Transaction {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tx := tm.pendingPool.Get(hash)
	if tx != nil {
		return tx
	}
	txc := tm.futurePool.Get(hash)
	if txc != nil {
		return txc.Transaction
	}
	return nil
}

// DelByAddressNonce del transaction specific addr-nonce transaction
// TODO Refactoring (addr, nonce) => Account
func (tm *TransactionManager) DelByAddressNonce(addr common.Address, nonce uint64) error {
	tm.mu.Lock()
	tm.pendingPool.Prune(addr, nonce)
	tm.futurePool.Prune(addr, nonce, nil)
	tm.mu.Unlock()

	tail := tm.canon.TailBlock()
	return tm.transit(tail, addr)
}

// TODO Need GETALL(?)
// // GetAll returns all transactions from transaction pool
// func (tm *TransactionManager) GetAll() []*Transaction {
// 	pending := tm.pendingPool.GetAll()
// 	future := tm.futurePool.GetAll()
// 	return append(pending, future...)
// }

// Len returns the number of transactions kept in the transaction manager.
func (tm *TransactionManager) Len() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.futurePool.Len() + tm.pendingPool.Len()
}

// Broadcast broadcasts transaction to network.
func (tm *TransactionManager) Broadcast(tx *Transaction) error {
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
