package core

import (
	"sync"

	"github.com/hashicorp/golang-lru"
	"github.com/medibloc/go-medibloc/common"
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
func (bp *BlockPool) Push(block *Block) error {
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

	bp.cache.Add(block.Hash(), lb)
	return nil
}

// Remove removes block in BlockPool.
func (bp *BlockPool) Remove(block *Block) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.cache.Remove(block.Hash())
}

// FindParent finds parent block.
func (bp *BlockPool) FindParent(block *Block) *Block {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	if plb := bp.findParentLinkedBlock(block); plb != nil {
		return plb.block
	}
	return nil
}

// FindUnlinkedAncestor finds block's unlinked ancestor in BlockPool.
func (bp *BlockPool) FindUnlinkedAncestor(block *Block) *Block {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	lb := bp.findParentLinkedBlock(block)
	if lb == nil {
		return block
	}

	for lb.parentLinkedBlock != nil {
		lb = lb.parentLinkedBlock
	}
	return lb.block
}

// FindChildren finds children blocks.
func (bp *BlockPool) FindChildren(block *Block) (childBlocks []*Block) {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	v, ok := bp.cache.Get(block.Hash())
	if !ok {
		return nil
	}

	lb := v.(*linkedBlock)
	for _, v := range lb.childLinkedBlocks {
		childBlocks = append(childBlocks, v.block)
	}
	return childBlocks
}

// Has returns true if BlockPool contains block.
func (bp *BlockPool) Has(block *Block) bool {
	return bp.cache.Contains(block.Hash())
}

func (bp *BlockPool) findParentLinkedBlock(block *Block) *linkedBlock {
	if plb, ok := bp.cache.Get(block.ParentHash()); ok {
		return plb.(*linkedBlock)
	}
	return nil
}

// TODO Improve lookup by adding another index of parent hash(?)
func (bp *BlockPool) findChildLinkedBlocks(block *Block) (childBlocks []*linkedBlock) {
	for _, key := range bp.cache.Keys() {
		v, ok := bp.cache.Get(key)
		if !ok {
			continue
		}

		lb := v.(*linkedBlock)
		if lb.block.Hash() == TODOTestGenesisBlock.Hash() {
			continue
		}
		if lb.block.ParentHash() == block.Hash() {
			childBlocks = append(childBlocks, lb)
		}
	}
	return childBlocks
}

type linkedBlock struct {
	block             *Block
	parentLinkedBlock *linkedBlock
	childLinkedBlocks map[common.Hash]*linkedBlock
}

func newLinkedBlock(block *Block) *linkedBlock {
	return &linkedBlock{
		block:             block,
		childLinkedBlocks: make(map[common.Hash]*linkedBlock),
	}
}

// Dispose cut the links. So, the block can be collected by GC.
func (lb *linkedBlock) dispose() {
	lb.block = nil
	if lb.parentLinkedBlock != nil {
		delete(lb.parentLinkedBlock.childLinkedBlocks, lb.block.Hash())
		lb.parentLinkedBlock = nil
	}
	for _, v := range lb.childLinkedBlocks {
		v.parentLinkedBlock = nil
	}
	lb.childLinkedBlocks = nil
}

func (lb *linkedBlock) linkParent(plb *linkedBlock) {
	plb.childLinkedBlocks[lb.block.Hash()] = lb
	lb.parentLinkedBlock = plb
}
