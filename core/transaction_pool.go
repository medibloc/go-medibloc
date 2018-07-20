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

	"sort"

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
			Data:  tx.String(),
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
	tx := cmpTx.(*sortable).Transaction

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

	// replace candidate
	candidate := bkt.peekFirst()
	pool.candidates.Del(from)
	pool.candidates.Set(from, &sortable{candidate})

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
	bkt := v.(*bucket)
	bkt.del(tx)

	// Remove from candidates
	pool.candidates.Del(from)

	// Remove bucket if empty
	if bkt.isEmpty() {
		return
	}
	pool.buckets.Set(from, bkt)

	// Replace candidate
	candidate := bkt.peekFirst()
	pool.candidates.Set(from, &sortable{candidate})
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

// comparable assigns a sequence to the transaction.
type comparable struct{ *Transaction }

func (tx *comparable) Less(o interface{}) bool { return tx.nonce < o.(*comparable).nonce }

type sortable struct{ *Transaction }

func (tx *sortable) Less(o interface{}) bool { return tx.timestamp < o.(*sortable).timestamp }

// transactions is sortable slice of comparable transactions.
type transactions []*comparable

func (t transactions) Len() int { return len(t) }

func (t transactions) Less(i, j int) bool { return t[i].Less(t[j]) }

func (t transactions) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// bucket is a set of transactions for each account.
type bucket struct {
	txs transactions
}

func newBucket() *bucket {
	return &bucket{
		txs: make(transactions, 0),
	}
}

func (b *bucket) Less(o interface{}) bool {
	return len(b.txs) > len(o.(*bucket).txs)
}

func (b *bucket) isEmpty() bool {
	return len(b.txs) == 0
}

func (b *bucket) push(tx *Transaction) {
	cmpTx := &comparable{tx}
	b.txs = append(b.txs, cmpTx)
	sort.Sort(b.txs)
}

func (b *bucket) peekFirst() *Transaction {
	if len(b.txs) == 0 {
		return nil
	}
	return b.txs[0].Transaction
}

func (b *bucket) peekLast() *Transaction {
	if len(b.txs) == 0 {
		return nil
	}
	return b.txs[len(b.txs)-1].Transaction
}

func (b *bucket) del(tx *Transaction) {
	for i, tt := range b.txs {
		if byteutils.Equal(tt.Transaction.Hash(), tx.Hash()) {
			b.txs = append(b.txs[:i], b.txs[i+1:]...)
			return
		}
	}
}
