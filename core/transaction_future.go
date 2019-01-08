package core

import (
	"sync"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/hashheap"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// FutureTransactionPool is a pool of all future transactions.
type FutureTransactionPool struct {
	mu sync.RWMutex

	size int
	buckets *hashheap.HashedHeap
	all map[string]*TxContext
}

// NewFutureTransactionPool returns FutureTransactionPool.
func NewFutureTransactionPool(size int) *FutureTransactionPool {
	return &FutureTransactionPool{
		size: size,
		buckets: hashheap.New(),
		all: make(map[string]*TxContext),
	}
}

func (pool *FutureTransactionPool) Get(hash []byte) *TxContext {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	tx, exist := pool.all[byteutils.Bytes2Hex(hash)]
	if !exist {
		return nil
	}
	return tx
}

func (pool *FutureTransactionPool) Set(tx *TxContext) (evicted *TxContext) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.exist(tx) {
		return nil
	}
	pool.all[byteutils.Bytes2Hex(tx.Hash())] = tx

	replaced := pool.set(tx)
	if replaced != nil {
		return replaced
	}

	evicted = pool.evict()
	if evicted != nil {
		return evicted
	}
	return nil
}

// TODO change input parameter to hash
func (pool *FutureTransactionPool) Del(tx *TxContext) (deleted *TxContext) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if !pool.exist(tx) {
		return nil
	}

	return pool.del(tx)
}

func (pool *FutureTransactionPool) Exist(tx *TxContext) bool {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.exist(tx)
}

func (pool *FutureTransactionPool) PopWithNonceUpperLimit(addr common.Address, nonceUpperLimit uint64) *TxContext {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	from := addr.Hex()
	v := pool.buckets.Del(from)
	if v == nil {
		return nil
	}
	bkt := v.(*bucket)

	tx := bkt.peekFirst()
	if tx == nil || tx.Nonce() > nonceUpperLimit {
		return nil
	}
	return pool.del(tx)
}

func (pool *FutureTransactionPool) Prune(addr common.Address, nonceLowerLimit uint64, deleteCallback func(tx *TxContext)) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	from := addr.Hex()
	v := pool.buckets.Del(from)
	if v == nil {
		return
	}
	bkt := v.(*bucket)

	for {
		tx := bkt.peekFirst()
		if tx == nil || tx.Nonce() >= nonceLowerLimit {
			break
		}
		pool.del(tx)
		if deleteCallback != nil {
			deleteCallback(tx)
		}
	}
}

func (pool *FutureTransactionPool) exist(tx *TxContext) bool  {
	_, exist := pool.all[byteutils.Bytes2Hex(tx.Hash())]
	return exist
}

func (pool *FutureTransactionPool) set(tx *TxContext) (evicted *TxContext) {
	from := tx.From().Hex()

	v := pool.buckets.Del(from)
	if v == nil {
		v = newBucket()
	}
	bkt := v.(*bucket)
	evicted = bkt.set(tx)
	pool.buckets.Set(from, bkt)
	return evicted
}

func (pool *FutureTransactionPool) evict() (evicted *TxContext) {
	if len(pool.all) <= pool.size {
		return nil
	}

	v := pool.buckets.Peek()
	if v == nil {
		return nil
	}
	bkt := v.(*bucket)

	tx := bkt.peekLast()
	if tx == nil {
		return nil
	}
	return pool.del(tx)
}

func (pool *FutureTransactionPool) del(tx *TxContext) (deleted *TxContext) {
	if !pool.exist(tx) {
		return nil
	}
	delete(pool.all, byteutils.Bytes2Hex(tx.Hash()))

	from := tx.From().Hex()

	v := pool.buckets.Del(from)
	if v == nil {
		return nil
	}
	bkt := v.(*bucket)
	deleted = bkt.del(tx.Nonce())
	if !bkt.isEmpty() {
		pool.buckets.Set(from, bkt)
	}
	return deleted
}

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

func (b *bucket) exist(nonce uint64) bool {
	return b.minTxs.Get(string(nonce)) != nil
}

func (b *bucket) set(tx *TxContext) (evicted *TxContext) {
	minTx := &minNonce{tx}
	maxTx := &maxNonce{tx}
	b.minTxs.Set(string(tx.Nonce()), minTx)
	v := b.maxTxs.Set(string(tx.Nonce()), maxTx)
	if v != nil {
		return v.(*maxNonce).TxContext
	}
	return nil
}

func (b *bucket) peekFirst() *TxContext {
	if b.minTxs.Len() == 0 {
		return nil
	}
	return b.minTxs.Peek().(*minNonce).TxContext
}

func (b *bucket) peekLast() *TxContext {
	if b.maxTxs.Len() == 0 {
		return nil
	}
	return b.maxTxs.Peek().(*maxNonce).TxContext
}

func (b *bucket) del(nonce uint64) (deleted *TxContext) {
	b.minTxs.Del(string(nonce))
	v := b.maxTxs.Del(string(nonce))
	if v != nil {
		return v.(*maxNonce).TxContext
	}
	return nil
}

// minNonce assigns a sequence by nonce to the transaction.
type minNonce struct{ *TxContext }

func (tx *minNonce) Less(o interface{}) bool { return tx.nonce < o.(*minNonce).nonce }

type maxNonce struct{ *TxContext }

func (tx *maxNonce) Less(o interface{}) bool { return tx.nonce > o.(*maxNonce).nonce }

