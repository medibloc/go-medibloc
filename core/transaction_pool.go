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
	"sync"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/hashheap"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// TransactionPool is a pool of all received transactions from network.
type TransactionPool struct {
	mu sync.RWMutex

	size int

	candidates *hashheap.HashedHeap
	buckets    *hashheap.HashedHeap
	all        map[string]*Transaction

	counter uint64

	eventEmitter *EventEmitter
}

// NewTransactionPool returns TransactionPool.
func NewTransactionPool(size int) *TransactionPool {
	return &TransactionPool{
		size:       size,
		candidates: hashheap.New(),
		buckets:    hashheap.New(),
		all:        make(map[string]*Transaction),
	}
}

// SetEventEmitter set emitter to transaction pool
func (pool *TransactionPool) SetEventEmitter(emitter *EventEmitter) {
	pool.eventEmitter = emitter
}

// Get returns transaction by tx hash.
func (pool *TransactionPool) Get(hash []byte) *Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	tx, ok := pool.all[byteutils.Bytes2Hex(hash)]
	if !ok {
		return nil
	}
	return tx
}

// GetAll returns all transactions in transaction pool
func (pool *TransactionPool) GetAll() []*Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	var txs []*Transaction

	if len(pool.all) == 0 {
		return nil
	}

	for _, tx := range pool.all {
		txs = append(txs, tx)
	}

	return txs
}

// GetByAddress returns transactions related to the address in transaction pool
func (pool *TransactionPool) GetByAddress(address common.Address) []*Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	var txs []*Transaction

	if len(pool.all) == 0 {
		return nil
	}

	for _, tx := range pool.all {
		if tx.IsRelatedToAddress(address) {
			txs = append(txs, tx)
		}
	}

	return txs
}

// Del deletes transaction.
func (pool *TransactionPool) Del(tx *Transaction) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.del(tx)
}

// Push pushes transaction to the pool.
func (pool *TransactionPool) Push(tx *Transaction) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if err := pool.push(tx); err != nil {
		return err
	}

	pool.evict()

	if pool.eventEmitter != nil {
		event := &Event{
			Topic: TopicPendingTransaction,
			Data:  byteutils.Bytes2Hex(tx.Hash()),
		}
		pool.eventEmitter.Trigger(event)
	}

	return nil
}

// Pop pop transaction from the pool.
func (pool *TransactionPool) Pop() *Transaction {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	cmpTx := pool.candidates.Peek()
	if cmpTx == nil {
		return nil
	}
	tx := cmpTx.(*ordered).Transaction

	pool.del(tx)

	return tx
}

func (pool *TransactionPool) push(tx *Transaction) error {
	// push to all
	if _, ok := pool.all[byteutils.Bytes2Hex(tx.Hash())]; ok {
		return ErrDuplicatedTransaction
	}
	pool.all[byteutils.Bytes2Hex(tx.Hash())] = tx

	from := tx.From().Str()

	// push to bucket
	var bkt *bucket
	v := pool.buckets.Get(from)
	if v != nil {
		pool.buckets.Del(from)
		bkt = v.(*bucket)
	} else {
		bkt = newBucket()
	}
	bkt.push(tx)
	pool.buckets.Set(from, bkt)

	pool.replaceCandidate(bkt, from)

	return nil
}

func (pool *TransactionPool) del(tx *Transaction) {
	// Remove from all
	if _, ok := pool.all[byteutils.Bytes2Hex(tx.Hash())]; !ok {
		return
	}
	delete(pool.all, byteutils.Bytes2Hex(tx.Hash()))

	from := tx.From().Str()

	// Remove from bucket
	v := pool.buckets.Get(from)
	if v == nil {
		logging.WithFields(logrus.Fields{
			"tx": tx,
		}).Error("Unable to find the bucket containing the transaction.")
		return
	}
	pool.buckets.Del(from)
	bkt := v.(*bucket)
	bkt.del(tx)
	if !bkt.isEmpty() {
		pool.buckets.Set(from, bkt)
	}

	pool.replaceCandidate(bkt, from)
}

func (pool *TransactionPool) replaceCandidate(bkt *bucket, addr string) {
	cand := bkt.peekFirst()
	if cand == nil {
		pool.candidates.Del(addr)
		return
	}

	v := pool.candidates.Get(addr)
	if v != nil && byteutils.Equal(cand.Hash(), v.(*ordered).Transaction.Hash()) {
		return
	}
	pool.candidates.Del(addr)
	pool.candidates.Set(addr, &ordered{Transaction: cand, ordering: pool.counter})
	pool.counter++
}

func (pool *TransactionPool) evict() {
	if len(pool.all) <= pool.size {
		return
	}

	// Peek longest bucket
	v := pool.buckets.Peek()
	if v == nil {
		return
	}
	bkt := v.(*bucket)

	tx := bkt.peekLast()
	if tx == nil {
		return
	}

	pool.del(tx)
}

type ordered struct {
	*Transaction
	ordering uint64
}

func (tx *ordered) Less(o interface{}) bool { return tx.ordering < o.(*ordered).ordering }

// minNonce assigns a sequence by nonce to the transaction.
type minNonce struct{ *Transaction }

func (tx *minNonce) Less(o interface{}) bool { return tx.nonce < o.(*minNonce).nonce }

type maxNonce struct{ *Transaction }

func (tx *maxNonce) Less(o interface{}) bool { return tx.nonce > o.(*maxNonce).nonce }

// bucket is a set of transactions for each account.
type bucket struct {
	minTxs *hashheap.HashedHeap
	maxTxs *hashheap.HashedHeap
}

func newBucket() *bucket {
	return &bucket{
		minTxs: hashheap.New(),
		maxTxs: hashheap.New(),
	}
}

func (b *bucket) Less(o interface{}) bool {
	return b.minTxs.Len() > o.(*bucket).minTxs.Len()
}

func (b *bucket) isEmpty() bool {
	return b.minTxs.Len() == 0
}

func (b *bucket) push(tx *Transaction) {
	minTx := &minNonce{tx}
	maxTx := &maxNonce{tx}
	b.minTxs.Set(byteutils.Bytes2Hex(tx.Hash()), minTx)
	b.maxTxs.Set(byteutils.Bytes2Hex(tx.Hash()), maxTx)
}

func (b *bucket) peekFirst() *Transaction {
	if b.minTxs.Len() == 0 {
		return nil
	}
	return b.minTxs.Peek().(*minNonce).Transaction
}

func (b *bucket) peekLast() *Transaction {
	if b.maxTxs.Len() == 0 {
		return nil
	}
	return b.maxTxs.Peek().(*maxNonce).Transaction
}

func (b *bucket) del(tx *Transaction) {
	b.minTxs.Del(byteutils.Bytes2Hex(tx.Hash()))
	b.maxTxs.Del(byteutils.Bytes2Hex(tx.Hash()))
}
