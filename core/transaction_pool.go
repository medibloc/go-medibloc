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
	"fmt"
	"sync"
	"time"

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
	all        map[string]*TransactionInPool
}

// NewTransactionPool returns TransactionPool.
func NewTransactionPool(size int) *TransactionPool {
	return &TransactionPool{
		size:       size,
		candidates: hashheap.New(),
		buckets:    hashheap.New(),
		all:        make(map[string]*TransactionInPool),
	}
}

// Get returns transaction by tx hash.
func (pool *TransactionPool) Get(hash []byte) *TransactionInPool {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	tp, ok := pool.all[byteutils.Bytes2Hex(hash)]
	if !ok {
		return nil
	}
	return tp
}

// GetAll returns all transactions in transaction pool
func (pool *TransactionPool) GetAll() []*Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	var txs []*Transaction

	if len(pool.all) == 0 {
		return nil
	}

	for _, v := range pool.all {
		txs = append(txs, v.Transaction)
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

	for _, v := range pool.all {
		if v.IsRelatedToAddress(address) {
			txs = append(txs, v.Transaction)
		}
	}

	return txs
}

// Del deletes transaction.
func (pool *TransactionPool) Del(hash []byte) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.del(hash)
}

// Push pushes transaction to the pool.
func (pool *TransactionPool) Push(tx *TransactionInPool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.push(tx)
}

// Pop pop transaction from the pool.
func (pool *TransactionPool) Pop() *TransactionInPool {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	cmpTx := pool.candidates.Peek()
	if cmpTx == nil {
		return nil
	}
	tx := cmpTx.(*TransactionInPool)

	pool.del(tx.Hash())

	return tx
}

func (pool *TransactionPool) push(tx *TransactionInPool) {
	old := pool.getByAddressAndNonce(tx.from, tx.nonce)
	if old != nil {
		delete(pool.all, old.HexHash())
	}
	pool.all[tx.HexHash()] = tx // always overwrite

	from := tx.From().Hex()

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
}

func (pool *TransactionPool) del(hash []byte) {
	// Remove from all
	tx, ok := pool.all[byteutils.Bytes2Hex(hash)]
	if !ok {
		return
	}
	delete(pool.all, byteutils.Bytes2Hex(hash))

	from := tx.From().Hex()

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
	if v != nil && byteutils.Equal(cand.Hash(), v.(*TransactionInPool).Transaction.Hash()) {
		return
	}
	pool.candidates.Del(addr)
	pool.candidates.Set(addr, cand)
}

//Evict remove transaction when pool is fulfilled.
func (pool *TransactionPool) Evict() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for {
		if len(pool.all) <= pool.size || pool.size < 0 { // 상수 선언
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

		pool.del(tx.hash)
	}
}

//LenByAddress returns number of transactions of specific address
func (pool *TransactionPool) LenByAddress(addr common.Address) int {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	v := pool.buckets.Get(addr.Hex())
	if v == nil {
		return 0
	}
	return v.(*bucket).len()
}

//PeekFirstByAddress returns transaction with lowest nonce among specific address's transactions
func (pool *TransactionPool) PeekFirstByAddress(addr common.Address) *TransactionInPool {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	v := pool.buckets.Get(addr.Hex())
	if v == nil {
		return nil
	}
	return v.(*bucket).peekFirst()
}

//PeekLastByAddress returns transaction with highest nonce among specific address's transactions
func (pool *TransactionPool) PeekLastByAddress(addr common.Address) *TransactionInPool {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	v := pool.buckets.Get(addr.Hex())
	if v == nil {
		return nil
	}
	return v.(*bucket).peekLast()
}

//GetByAddressAndNonce returns transaction with addr-nonce
func (pool *TransactionPool) GetByAddressAndNonce(addr common.Address, nonce uint64) *TransactionInPool {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return pool.getByAddressAndNonce(addr, nonce)
}

func (pool *TransactionPool) getByAddressAndNonce(addr common.Address, nonce uint64) *TransactionInPool {
	v := pool.buckets.Get(addr.Hex())
	if v == nil {
		return nil
	}
	key := addrNoncePair(addr, nonce)
	return v.(*bucket).getByAddrNonce(key)
}

//TransactionInPool is structure to handle transaction in pool
type TransactionInPool struct {
	*Transaction
	executable      ExecutableTx
	incomeTimestamp int64
	bandwidth       *Bandwidth
}

//IncomeTimestamp returns income timestamp
func (tx *TransactionInPool) IncomeTimestamp() int64 {
	return tx.incomeTimestamp
}

//SetIncomeTimestamp sets income timestamp
func (tx *TransactionInPool) SetIncomeTimestamp(incomeTimestamp int64) {
	tx.incomeTimestamp = incomeTimestamp
}

//NewTransactionInPool returns new TransactionInPool with current timestamp
func NewTransactionInPool(transaction *Transaction, txMap TxFactory) (*TransactionInPool, error) {
	tx, err := transaction.Executable(txMap)
	if err != nil {
		return nil, err
	}
	return &TransactionInPool{
		Transaction:     transaction,
		executable:      tx,
		incomeTimestamp: time.Now().UnixNano(),
		bandwidth:       NewBandwidth(tx.Bandwidth()),
	}, nil
}

//Less returns if it has earlier timestamp
func (tx *TransactionInPool) Less(o interface{}) bool {
	return tx.incomeTimestamp < o.(*TransactionInPool).incomeTimestamp
}

// minNonce assigns a sequence by nonce to the transaction.
type minNonce struct{ *TransactionInPool }

func (tx *minNonce) Less(o interface{}) bool { return tx.nonce < o.(*minNonce).nonce }

type maxNonce struct{ *TransactionInPool }

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

func (b *bucket) len() int {
	return b.minTxs.Len()
}

func (b *bucket) isEmpty() bool {
	return b.minTxs.Len() == 0
}

func (b *bucket) push(tx *TransactionInPool) {
	minTx := &minNonce{tx}
	maxTx := &maxNonce{tx}
	key := addrNoncePair(tx.from, tx.nonce)
	b.minTxs.Set(key, minTx)
	b.maxTxs.Set(key, maxTx)
}

func (b *bucket) peekFirst() *TransactionInPool {
	if b.minTxs.Len() == 0 {
		return nil
	}
	return b.minTxs.Peek().(*minNonce).TransactionInPool
}

func (b *bucket) peekLast() *TransactionInPool {
	if b.maxTxs.Len() == 0 {
		return nil
	}
	return b.maxTxs.Peek().(*maxNonce).TransactionInPool
}

func (b *bucket) del(tx *TransactionInPool) {
	key := addrNoncePair(tx.from, tx.nonce)
	b.minTxs.Del(key)
	b.maxTxs.Del(key)
}

func (b *bucket) getByAddrNonce(pair string) *TransactionInPool {
	v := b.minTxs.Get(pair)
	if v == nil {
		return nil
	}
	return v.(*minNonce).TransactionInPool
}

func addrNoncePair(addr common.Address, nonce uint64) string {
	return fmt.Sprintf("%v-%v", addr.Hex(), nonce)
}
