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

	lru "github.com/hashicorp/golang-lru"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

//BlockPool is a pool of all received blocks from network.
type BlockPool struct {
	cache *lru.Cache
	mu    sync.RWMutex
	size  int
}

// NewBlockPool returns BlockPool.
func NewBlockPool(size int) (bp *BlockPool, err error) {
	bp = &BlockPool{
		size: size,
	}
	bp.cache, err = lru.NewWithEvict(size, func(key interface{}, value interface{}) {
		lb := value.(*linkedBlock)
		if lb != nil {
			lb.dispose()
		}
	})
	if err != nil {
		logging.WithFields(logrus.Fields{
			"size": size,
			"err":  err,
		}).Error("Failed to initialize lru cache.")
		return nil, err
	}
	return bp, nil
}

// Push links the block with parent and children blocks and push to the BlockPool.
func (bp *BlockPool) Push(block HashableBlock) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if block == nil {
		return ErrNilArgument
	}

	if bp.Has(block) {
		logging.WithFields(logrus.Fields{
			"block": block,
		}).Debug("Found duplicated block.")
		return ErrDuplicatedBlock
	}

	lb := newLinkedBlock(block)

	if plb := bp.findParentLinkedBlock(block); plb != nil {
		lb.linkParent(plb)
	}

	for _, clb := range bp.findChildLinkedBlocks(block) {
		clb.linkParent(lb)
	}

	bp.cache.Add(byteutils.Bytes2Hex(block.Hash()), lb)
	return nil
}

// Remove removes block in BlockPool.
func (bp *BlockPool) Remove(block HashableBlock) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.cache.Remove(byteutils.Bytes2Hex(block.Hash()))
}

// FindParent finds parent block.
func (bp *BlockPool) FindParent(block HashableBlock) HashableBlock {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	if plb := bp.findParentLinkedBlock(block); plb != nil {
		return plb.block
	}
	return nil
}

// FindUnlinkedAncestor finds block's unlinked ancestor in BlockPool.
func (bp *BlockPool) FindUnlinkedAncestor(block HashableBlock) HashableBlock {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	lb := bp.findParentLinkedBlock(block)
	if lb == nil {
		return nil
	}

	for lb.parentLinkedBlock != nil {
		lb = lb.parentLinkedBlock
	}
	return lb.block
}

// FindChildren finds children blocks.
func (bp *BlockPool) FindChildren(block HashableBlock) (childBlocks []HashableBlock) {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	// Found parameter block.
	if v, ok := bp.cache.Get(byteutils.Bytes2Hex(block.Hash())); ok {
		lb := v.(*linkedBlock)
		for _, vv := range lb.childLinkedBlocks {
			childBlocks = append(childBlocks, vv.block)
		}
		return childBlocks
	}

	// Not found parameter block.
	children := bp.findChildLinkedBlocks(block)
	for _, v := range children {
		childBlocks = append(childBlocks, v.block)
	}
	return childBlocks
}

// Has returns true if BlockPool contains block.
func (bp *BlockPool) Has(block HashableBlock) bool {
	return bp.cache.Contains(byteutils.Bytes2Hex(block.Hash()))
}

func (bp *BlockPool) findParentLinkedBlock(block HashableBlock) *linkedBlock {
	if plb, ok := bp.cache.Get(byteutils.Bytes2Hex(block.ParentHash())); ok {
		return plb.(*linkedBlock)
	}
	return nil
}

// TODO Improve lookup by adding another index of parent hash(?)
func (bp *BlockPool) findChildLinkedBlocks(block HashableBlock) (childBlocks []*linkedBlock) {
	for _, key := range bp.cache.Keys() {
		v, ok := bp.cache.Get(key)
		if !ok {
			continue
		}

		lb := v.(*linkedBlock)
		if byteutils.Equal(lb.block.Hash(), GenesisHash) {
			continue
		}
		if byteutils.Equal(lb.block.ParentHash(), block.Hash()) {
			childBlocks = append(childBlocks, lb)
		}
	}
	return childBlocks
}

type linkedBlock struct {
	block             HashableBlock
	parentLinkedBlock *linkedBlock
	childLinkedBlocks map[string]*linkedBlock
}

func newLinkedBlock(block HashableBlock) *linkedBlock {
	return &linkedBlock{
		block:             block,
		childLinkedBlocks: make(map[string]*linkedBlock),
	}
}

// Dispose cut the links. So, the block can be collected by GC.
func (lb *linkedBlock) dispose() {
	if plb := lb.parentLinkedBlock; plb != nil {
		lb.unlinkParent(plb)
	}
	for _, clb := range lb.childLinkedBlocks {
		clb.unlinkParent(lb)
	}
	lb.childLinkedBlocks = nil
	lb.block = nil
}

func (lb *linkedBlock) linkParent(plb *linkedBlock) {
	plb.childLinkedBlocks[byteutils.Bytes2Hex(lb.block.Hash())] = lb
	lb.parentLinkedBlock = plb
}

func (lb *linkedBlock) unlinkParent(plb *linkedBlock) {
	delete(plb.childLinkedBlocks, byteutils.Bytes2Hex(lb.block.Hash()))
	lb.parentLinkedBlock = nil
}
