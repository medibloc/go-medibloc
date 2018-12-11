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

	"github.com/medibloc/go-medibloc/common/hashheap"
)

// GappedTxPool is a pool of transactions which has gap between current nonce.
type GappedTxPool struct {
	mu sync.RWMutex

	size int

	//candidates *hashheap.HashedHeap
	buckets    *hashheap.HashedHeap
	//all        map[string]*Transaction

	counter uint64

	eventEmitter *EventEmitter
}

// Push pushes transaction to the gapped tx pool.
func (gp *GappedTxPool) Push(tx *Transaction) error {
	gp.mu.Lock()
	defer gp.mu.Unlock()

	from := tx.From().Str()

	// push to bucket
	var bkt *bucket
	v := gp.buckets.Get(from)
	if v != nil {
		gp.buckets.Del(from) // del no required? @jiseob
		bkt = v.(*bucket)
	} else {
		bkt = newBucket()
	}
	// TODO check max tx limit
	bkt.push(tx)
	gp.buckets.Set(from, bkt)

	return nil
}

// ContinuousTx return PopContinuousTxs.
func (gp *GappedTxPool) PopContinuousTxs(tx *Transaction) []*Transaction {
	gp.mu.Lock()
	defer gp.mu.Unlock()

	var Txs []*Transaction
	from := tx.From().Str()
	nonce := tx.nonce

	v := gp.buckets.Get(from)
	if v == nil {
		return nil
	}
	bkt := v.(*bucket)
	for {
		minTx := bkt.peekFirst()
		if minTx != nil && (nonce + 1 == minTx.nonce) {
			nonce ++
			bkt.minTxs.Pop()
			Txs = append(Txs, minTx)
			continue
		}
		break
	}
	return Txs
}