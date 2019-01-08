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
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/event"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
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
	bandwidthInfo map[common.Address]*common.Bandwidth // how much bandwidth used by transactions payed by payer in pending pool
}

// NewTransactionManager create a new TransactionManager.
func NewTransactionManager(cfg *medletpb.Config) *TransactionManager {
	return &TransactionManager{
		chainID:           cfg.Global.ChainId,
		receivedMessageCh: make(chan net.Message, defaultTransactionMessageChanSize),
		quitCh:            make(chan int),
		pendingPool:       NewTransactionPool(-1),
		futurePool:        NewTransactionPool(int(cfg.Chain.TransactionPoolSize)),
		bandwidthInfo:     make(map[common.Address]*common.Bandwidth),
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
func (tm *TransactionManager) InjectEmitter(emitter *event.Emitter) {
	tm.pendingPool.SetEventEmitter(emitter)
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
func (tm *TransactionManager) PushAndBroadcast(transactions ...*coreState.Transaction) (failed map[string]error) {
	failed = make(map[string]error)
	addrs, _, dropped := tm.pushToFuturePool(transactions...)
	for k, v := range dropped {
		failed[k] = v
	}
	pended, dropped := tm.transitTxs(tm.bm.bc.MainTailBlock().State(), addrs...)
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
func (tm *TransactionManager) PushAndExclusiveBroadcast(transactions ...*coreState.Transaction) (failed map[string]error) {
	failed = make(map[string]error)
	addrs, exclusiveFilter, dropped := tm.pushToFuturePool(transactions...)
	for k, v := range dropped {
		failed[k] = v
	}
	pended, dropped := tm.transitTxs(tm.bm.bc.MainTailBlock().State(), addrs...)
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

func (tm *TransactionManager) pushToFuturePool(transactions ...*coreState.Transaction) (addrs []common.Address, future map[string]bool, dropped map[string]error) {
	future = make(map[string]bool)
	dropped = make(map[string]error)
	addrMap := make(map[common.Address]int)

	for _, transaction := range transactions {
		if tm.pendingPool.Get(transaction.Hash()) != nil || tm.futurePool.Get(transaction.Hash()) != nil {
			dropped[transaction.HexHash()] = ErrDuplicatedTransaction
			continue
		}
		tm.bm.bc.eventEmitter.Register()
		if err := transaction.VerifyIntegrity(tm.chainID); err != nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": transaction,
				"err":         err,
			}).Debug("Failed to verify transaction.")
			dropped[transaction.HexHash()] = err
			continue
		}

		tp, err := NewTransactionContext(transaction)
		if err != nil {
			dropped[transaction.HexHash()] = err
			continue
		}

		// add to future pool
		tm.futurePool.Push(tp)
		tm.futurePool.Evict()
		addrMap[transaction.From()]++
		future[transaction.HexHash()] = true
	}

	addrs = make([]common.Address, 0, len(addrMap))
	for addr := range addrMap {
		addrs = append(addrs, addr)
	}
	return addrs, future, dropped
}

func (tm *TransactionManager) transitTxs(bs *BlockState, addrs ...common.Address) (pended []*coreState.Transaction, dropped map[string]error) {
	pended = make([]*coreState.Transaction, 0)
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

func (tm *TransactionManager) addToPendingPool(bs *BlockState, tx *TransactionContext) error {
	// append case (first item pending)
	tm.futurePool.Del(tx.Hash())
	if err := tm.addBandwidthInfo(bs, tx); err != nil {
		return err
	}
	tm.pendingPool.Push(tx)
	return nil
}

func (tm *TransactionManager) replaceBandwidthInfo(bs *BlockState, new, old *TransactionContext) error {
	if new == nil || old == nil {
		return ErrNilArgument
	}

	tm.bwInfoMu.Lock()
	defer tm.bwInfoMu.Unlock()

	existing := tm.bandwidthInfo[new.Payer()].Clone()
	if new.Payer().Equals(old.Payer()) {
		existing.Sub(old.Bandwidth())
	}

	if err := tm.verifyPayerPoints(bs, new, existing); err != nil {
		return err
	}

	_, ok := tm.bandwidthInfo[new.Payer()]
	if !ok {
		tm.bandwidthInfo[new.Payer()] = common.NewBandwidth(0, 0)
	}
	tm.bandwidthInfo[new.Payer()].Add(new.Bandwidth())

	tm.bandwidthInfo[old.Payer()].Sub(old.Bandwidth())
	if tm.bandwidthInfo[old.Payer()].IsZero() {
		delete(tm.bandwidthInfo, old.Payer())
	}
	if _, ok := tm.bandwidthInfo[new.Payer()]; !ok {
		tm.bandwidthInfo[new.Payer()] = common.NewBandwidth(0, 0)
	}
	tm.bandwidthInfo[new.Payer()].Add(new.Bandwidth())
	return nil
}

func (tm *TransactionManager) addBandwidthInfo(bs *BlockState, new *TransactionContext) error {
	if new == nil {
		return ErrNilArgument
	}

	tm.bwInfoMu.Lock()
	defer tm.bwInfoMu.Unlock()

	existing, ok := tm.bandwidthInfo[new.Payer()]
	if !ok {
		existing = common.NewBandwidth(0, 0)
	}
	if err := tm.verifyPayerPoints(bs, new, existing); err != nil {
		return err
	}

	if _, ok := tm.bandwidthInfo[new.Payer()]; !ok {
		tm.bandwidthInfo[new.Payer()] = common.NewBandwidth(0, 0)
	}
	tm.bandwidthInfo[new.Payer()].Add(new.Bandwidth())
	return nil
}

func (tm *TransactionManager) verifyPayerPoints(bs *BlockState, transaction *TransactionContext, existing *common.Bandwidth) error {
	payerAcc, err := bs.GetAccount(transaction.Payer())
	if err != nil {
		return err
	}

	if err := transaction.CheckAccountPoints(payerAcc, bs.Price(), existing); err != nil {
		return err
	}
	return nil
}

// Pop pop transaction from TransactionManager.
func (tm *TransactionManager) Pop() *transaction.ExecutableTx {
	tx := tm.pendingPool.Pop()
	if tx == nil {
		return nil
	}
	tm.bwInfoMu.Lock()
	tm.bandwidthInfo[tx.Payer()].Sub(tx.Bandwidth())
	if tm.bandwidthInfo[tx.Payer()].IsZero() {
		delete(tm.bandwidthInfo, tx.Payer())
	}
	tm.bwInfoMu.Unlock()

	success, _ := tm.transitTxs(tm.bm.bc.mainTailBlock.State(), tx.From())
	for _, t := range success {
		err := tm.Broadcast(t)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": t,
				"err":         err,
			}).Debug("failed to broadcast transaction")
		}
	}
	return tx.ExecutableTx
}

// Get transaction from transaction pool.
func (tm *TransactionManager) Get(hash []byte) *coreState.Transaction {
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
		if tx == nil || nonce < tx.Nonce() {
			break
		}
		tm.bwInfoMu.Lock()
		tm.bandwidthInfo[tx.Payer()].Sub(tx.Bandwidth())
		if tm.bandwidthInfo[tx.Payer()].IsZero() {
			delete(tm.bandwidthInfo, tx.Payer())
		}
		tm.bwInfoMu.Unlock()
		tm.pendingPool.Del(tx.Hash())
	}
	for {
		tx := tm.futurePool.PeekFirstByAddress(addr)
		if tx == nil || nonce < tx.Nonce() {
			break
		}
		tm.futurePool.Del(tx.Hash())
	}
	success, _ := tm.transitTxs(tm.bm.bc.mainTailBlock.State(), addr)
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
func (tm *TransactionManager) GetAll() []*coreState.Transaction {
	pending := tm.pendingPool.GetAll()
	future := tm.futurePool.GetAll()
	return append(pending, future...)
}

// Broadcast broadcasts transaction to network.
func (tm *TransactionManager) Broadcast(tx *coreState.Transaction) error {
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

func txFromNetMsg(msg net.Message) (*coreState.Transaction, error) {
	if msg.MessageType() != MessageTypeNewTx {
		logging.WithFields(logrus.Fields{
			"type": msg.MessageType(),
			"msg":  msg,
		}).Debug("Received unregistered message.")
		return nil, errors.New("invalid message type")
	}

	tx := new(coreState.Transaction)
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
