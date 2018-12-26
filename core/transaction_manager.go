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
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Parameters for transaction manager
var (
	MaxPending                        = 64
	AllowReplacePendingDuration       = 10 * time.Minute
	defaultTransactionMessageChanSize = 128
)

// TransactionManager manages transactions' pool and network service.
type TransactionManager struct {
	chainID uint32

	receivedMessageCh chan net.Message
	quitCh            chan int
	bm                *BlockManager
	ns                net.Service

	pendingPool *TransactionPool
	futurePool  *TransactionPool

	bwInfoMu      sync.Mutex
	bandwidthInfo map[common.Address]*Bandwidth // how much bandwidth used by transactions payed by payer in pending pool

	eventEmitter *EventEmitter
}

// NewTransactionManager create a new TransactionManager.
func NewTransactionManager(cfg *medletpb.Config) *TransactionManager {
	return &TransactionManager{
		chainID:           cfg.Global.ChainId,
		receivedMessageCh: make(chan net.Message, defaultTransactionMessageChanSize),
		quitCh:            make(chan int),
		pendingPool:       NewTransactionPool(-1),
		futurePool:        NewTransactionPool(int(cfg.Chain.TransactionPoolSize)),
		bandwidthInfo:     make(map[common.Address]*Bandwidth),
		ns:                nil,
	}
}

// Setup sets up TransactionManager.
func (tm *TransactionManager) Setup(bm *BlockManager, ns net.Service) {
	if ns != nil {
		tm.bm = bm
		tm.ns = ns
		tm.registerInNetwork()
	}
}

// InjectEmitter inject emitter generated from medlet to transaction manager
func (tm *TransactionManager) InjectEmitter(emitter *EventEmitter) {
	tm.eventEmitter = emitter
}

// Start starts TransactionManager.
func (tm *TransactionManager) Start() {
	logging.Console().WithFields(logrus.Fields{
		"MaxPending":     MaxPending,
		"futurePoolSize": tm.futurePool.size,
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

//PushAndBroadcast push and broad all transactions that moved from future to pending pool
func (tm *TransactionManager) PushAndBroadcast(transactions ...*Transaction) (failed map[string]error) {
	failed = make(map[string]error)
	addrs, _, dropped := tm.pushToFuturePool(transactions...)
	for k, v := range dropped {
		failed[k] = v
	}
	pended, dropped := tm.transitTxs(tm.bm.cm.MainTailBlock().State(), addrs...)
	for k, v := range dropped {
		failed[k] = v
	}

	for _, transaction := range pended {
		err := tm.Broadcast(transaction)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": transaction,
				"err":         err,
			}).Debug("failed to broadcast transaction")
		}
	}

	for hash, err := range failed {
		logging.Console().WithFields(logrus.Fields{
			"txHash": hash,
			"err":    err,
		}).Debug("dropped during tx transition")
	}
	return failed
}

// PushAndExclusiveBroadcast pushes transactions to transaction manager and broadcast only previously exist in future pool
func (tm *TransactionManager) PushAndExclusiveBroadcast(transactions ...*Transaction) (failed map[string]error) {
	failed = make(map[string]error)
	addrs, exclusiveFilter, dropped := tm.pushToFuturePool(transactions...)
	for k, v := range dropped {
		failed[k] = v
	}
	pended, dropped := tm.transitTxs(tm.bm.cm.MainTailBlock().State(), addrs...)
	for k, v := range dropped {
		failed[k] = v
	}

	for _, transaction := range pended {
		if _, ok := exclusiveFilter[transaction.HexHash()]; ok {
			continue
		}
		err := tm.Broadcast(transaction)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": transaction,
				"err":         err,
			}).Debug("failed to broadcast transaction")
		}
	}

	for hash, err := range failed {
		logging.Console().WithFields(logrus.Fields{
			"txHash": hash,
			"err":    err,
		}).Debug("dropped during tx transition ")
	}
	return failed
}

func (tm *TransactionManager) pushToFuturePool(transactions ...*Transaction) (addrs []common.Address, future map[string]bool, dropped map[string]error) {
	future = make(map[string]bool)
	dropped = make(map[string]error)
	addrMap := make(map[common.Address]int)

	for _, transaction := range transactions {
		if tm.pendingPool.Get(transaction.hash) != nil || tm.futurePool.Get(transaction.hash) != nil {
			dropped[transaction.HexHash()] = ErrDuplicatedTransaction
			continue
		}
		tm.eventEmitter.Register()
		if err := transaction.VerifyIntegrity(tm.chainID); err != nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": transaction,
				"err":         err,
			}).Debug("Failed to verify transaction.")
			dropped[transaction.HexHash()] = err
			continue
		}

		tp, err := NewTransactionInPool(transaction, tm.bm.txMap)
		if err != nil {
			dropped[transaction.HexHash()] = err
			continue
		}

		// add to future pool
		tm.futurePool.Push(tp)
		if tm.eventEmitter != nil {
			tp.TriggerEvent(tm.eventEmitter, TopicFutureTransaction)
			tp.TriggerAccEvent(tm.eventEmitter, TypeAccountTransactionFuture)
		}
		tm.futurePool.Evict()
		addrMap[transaction.from]++
		future[transaction.HexHash()] = true
	}

	addrs = make([]common.Address, 0, len(addrMap))
	for addr := range addrMap {
		addrs = append(addrs, addr)
	}
	return addrs, future, dropped
}

func (tm *TransactionManager) transitTxs(bs *BlockState, addrs ...common.Address) (pended []*Transaction, dropped map[string]error) {
	pended = make([]*Transaction, 0)
	dropped = make(map[string]error)

	for _, addr := range addrs {
		for {
			firstInFuture := tm.futurePool.PeekFirstByAddress(addr)
			if firstInFuture == nil {
				break // no tx in acc
			}
			acc, err := bs.GetAccount(addr)
			if err != nil {
				tm.futurePool.Del(firstInFuture.Hash())
				dropped[firstInFuture.HexHash()] = err
				continue
			}
			if firstInFuture.Nonce() <= acc.Nonce {
				tm.futurePool.Del(firstInFuture.Hash())
				dropped[firstInFuture.HexHash()] = ErrSmallTransactionNonce
				continue
			}

			lastInPending := tm.pendingPool.PeekLastByAddress(addr)
			if lastInPending == nil { // pending is empty
				if acc.Nonce+1 != firstInFuture.Nonce() {
					break // nonce gap exist
				}
				if err := tm.addToPendingPool(bs, firstInFuture); err != nil {
					dropped[firstInFuture.HexHash()] = err
					continue
				}
				pended = append(pended, firstInFuture.Transaction)
				continue
			}
			if lastInPending.Nonce()+1 < firstInFuture.Nonce() {
				break // transaction nonce gap exist
			}
			if firstInFuture.Nonce() <= lastInPending.Nonce() {
				tm.futurePool.Del(firstInFuture.Hash())
				old := tm.pendingPool.GetByAddressAndNonce(addr, firstInFuture.Nonce())
				if old == nil {
					// append case
					if err := tm.addToPendingPool(bs, firstInFuture); err != nil {
						dropped[firstInFuture.HexHash()] = err
						continue
					}
					pended = append(pended, firstInFuture.Transaction)
					continue
				}
				if time.Now().Sub(time.Unix(0, old.incomeTimestamp)) < AllowReplacePendingDuration {
					dropped[firstInFuture.HexHash()] = ErrFailedToReplacePendingTx
					continue
				}
				// replace case
				if err := tm.replaceBandwidthInfo(bs, firstInFuture, old); err != nil {
					dropped[firstInFuture.HexHash()] = err
					continue
				}
				tm.pendingPool.Push(firstInFuture)
				if tm.eventEmitter != nil {
					firstInFuture.TriggerEvent(tm.eventEmitter, TopicPendingTransaction)
					firstInFuture.TriggerAccEvent(tm.eventEmitter, TypeAccountTransactionPending)
				}

				pended = append(pended, firstInFuture.Transaction)
				continue
			}
			if MaxPending <= tm.pendingPool.LenByAddress(addr) {
				break // pending bucket is full
			}
			// append case
			if err := tm.addToPendingPool(bs, firstInFuture); err != nil {
				dropped[firstInFuture.HexHash()] = err
				continue
			}
			pended = append(pended, firstInFuture.Transaction)
		}
	}
	return pended, dropped
}

func (tm *TransactionManager) addToPendingPool(bs *BlockState, tx *TransactionInPool) error {
	// append case (first item pending)
	tm.futurePool.Del(tx.Hash())
	if err := tm.addBandwidthInfo(bs, tx); err != nil {
		return err
	}
	tm.pendingPool.Push(tx)
	if tm.eventEmitter != nil {
		tx.TriggerEvent(tm.eventEmitter, TopicPendingTransaction)
		tx.TriggerAccEvent(tm.eventEmitter, TypeAccountTransactionPending)
	}

	return nil
}

func (tm *TransactionManager) replaceBandwidthInfo(bs *BlockState, new, old *TransactionInPool) error {
	if new == nil || old == nil {
		return ErrNilArgument
	}

	tm.bwInfoMu.Lock()
	defer tm.bwInfoMu.Unlock()

	newBandwidthInfo, ok := tm.bandwidthInfo[new.payer]
	if !ok {
		newBandwidthInfo = NewBandwidth(0, 0)
	}
	if new.payer.Equals(old.payer) {
		newBandwidthInfo.Sub(old.bandwidth)
	}
	newBandwidthInfo.Add(new.bandwidth)
	points, err := newBandwidthInfo.CalcPoints(bs.Price())
	if err != nil {
		return err
	}
	if err := tm.verifyPayerPoints(bs, new.payer, points, new.Transaction); err != nil {
		return err
	}

	tm.bandwidthInfo[old.payer].Sub(old.bandwidth)
	if tm.bandwidthInfo[old.payer].IsZero() {
		delete(tm.bandwidthInfo, old.payer)
	}
	if _, ok := tm.bandwidthInfo[new.payer]; !ok {
		tm.bandwidthInfo[new.payer] = NewBandwidth(0, 0)
	}
	tm.bandwidthInfo[new.payer].Add(new.bandwidth)
	return nil
}

func (tm *TransactionManager) addBandwidthInfo(bs *BlockState, new *TransactionInPool) error {
	if new == nil {
		return ErrNilArgument
	}

	tm.bwInfoMu.Lock()
	defer tm.bwInfoMu.Unlock()

	newBandwidthInfo, ok := tm.bandwidthInfo[new.payer]
	if !ok {
		newBandwidthInfo = NewBandwidth(0, 0)
	}

	newBandwidthInfo.Add(new.bandwidth)
	points, err := newBandwidthInfo.CalcPoints(bs.Price())
	if err != nil {
		return err
	}
	if err := tm.verifyPayerPoints(bs, new.payer, points, new.Transaction); err != nil {
		return err
	}

	if _, ok := tm.bandwidthInfo[new.payer]; !ok {
		tm.bandwidthInfo[new.payer] = NewBandwidth(0, 0)
	}
	tm.bandwidthInfo[new.payer].Add(new.bandwidth)
	return nil
}

func (tm *TransactionManager) verifyPayerPoints(bs *BlockState, payer common.Address, points *util.Uint128, transaction *Transaction) error {
	payerAcc, err := bs.GetAccount(payer)
	if err != nil {
		return err
	}
	if err := payerAcc.UpdatePoints(time.Now().Unix()); err != nil {
		return err
	}
	if err := payerAcc.checkAccountPoints(transaction, points); err != nil {
		return err
	}
	return nil
}

// Pop pop transaction from TransactionManager.
func (tm *TransactionManager) Pop() *Transaction {
	tx := tm.pendingPool.Pop()
	if tx == nil {
		return nil
	}
	tm.bwInfoMu.Lock()
	tm.bandwidthInfo[tx.payer].Sub(tx.bandwidth)
	if tm.bandwidthInfo[tx.payer].IsZero() {
		delete(tm.bandwidthInfo, tx.payer)
	}
	tm.bwInfoMu.Unlock()

	success, _ := tm.transitTxs(tm.bm.cm.mainTailBlock.State(), tx.From())
	for _, t := range success {
		err := tm.Broadcast(t)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": t,
				"err":         err,
			}).Debug("failed to broadcast transaction")
		}
	}
	return tx.Transaction
}

// Get transaction from transaction pool.
func (tm *TransactionManager) Get(hash []byte) *Transaction {
	v := tm.pendingPool.Get(hash)
	if v != nil {
		return v.Transaction
	}
	v = tm.futurePool.Get(hash)
	if v != nil {
		return v.Transaction
	}
	return nil
}

//DelByAddressNonce del transaction specific addr-nonce transaction
func (tm *TransactionManager) DelByAddressNonce(addr common.Address, nonce uint64) {
	for {
		tx := tm.pendingPool.PeekFirstByAddress(addr)
		if tx == nil || nonce < tx.nonce {
			break
		}
		tm.bwInfoMu.Lock()
		tm.bandwidthInfo[tx.payer].Sub(tx.bandwidth)
		if tm.bandwidthInfo[tx.payer].IsZero() {
			delete(tm.bandwidthInfo, tx.payer)
		}
		tm.bwInfoMu.Unlock()
		tm.pendingPool.Del(tx.hash)
	}
	for {
		tx := tm.futurePool.PeekFirstByAddress(addr)
		if tx == nil || nonce < tx.nonce {
			break
		}
		tm.futurePool.Del(tx.hash)
	}
	success, _ := tm.transitTxs(tm.bm.cm.mainTailBlock.State(), addr)
	for _, t := range success {
		err := tm.Broadcast(t)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": t,
				"err":         err,
			}).Debug("failed to broadcast transaction")
		}
	}
}

// GetAll returns all transactions from transaction pool
func (tm *TransactionManager) GetAll() []*Transaction {
	pending := tm.pendingPool.GetAll()
	future := tm.futurePool.GetAll()
	return append(pending, future...)
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
