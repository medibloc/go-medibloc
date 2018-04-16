// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

const (
	cacheSize             = 128
	tailBlockKeyInStorage = "tailBlock"
)

// BlockChain manages blockchain structure.
type BlockChain struct {
	cachedBlocks  *lru.Cache
	chainID       uint32
	mainTailBlock *Block
	mu            sync.RWMutex
	// tailBlocks all tail blocks including mainTailBlock
	tailBlocks *lru.Cache
	storage    storage.Storage
}

// NewBlockChain return new BlockChain instance
func NewBlockChain(chainID uint32, genesisBlock *Block, storage storage.Storage) (*BlockChain, error) {
	bc := &BlockChain{
		chainID:       chainID,
		mainTailBlock: genesisBlock,
		storage:       storage,
	}
	if v, err := storage.Get([]byte(tailBlockKeyInStorage)); err == nil {
		blockData, err := bytesToBlockData(v)
		if err != nil {
			return nil, err
		}
		bc.mainTailBlock, err = blockData.GetExecutedBlock(storage)
		if err != nil {
			return nil, err
		}
	}
	var err error
	bc.cachedBlocks, err = lru.New(cacheSize)
	if err != nil {
		return nil, err
	}
	bc.tailBlocks, err = lru.New(cacheSize)
	if err != nil {
		return nil, err
	}
	addBlockToCache(genesisBlock, bc.cachedBlocks)
	addBlockToCache(genesisBlock, bc.tailBlocks)
	return bc, nil
}

// ChainID returns ChainID.
func (bc *BlockChain) ChainID() uint32 {
	return bc.chainID
}

// GetBlock returns GetBlock.
func (bc *BlockChain) GetBlock(hash common.Hash) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if v, ok := bc.cachedBlocks.Get(hash.Str()); ok {
		return v.(*Block)
	}
	// Get BlockData from storage (may be deleted unwillingly by LRU cache)
	bytes, err := bc.storage.Get(hash.Bytes())
	if err == nil {
		blockData, err := bytesToBlockData(bytes)
		if err != nil {
			logging.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("failed to parse stored block data %s", bytes)
			return nil
		}
		block, err := blockData.GetExecutedBlock(bc.storage)
		if err != nil {
			logging.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("failed to recover stored block %s", bytes)
			return nil
		}
		return block
	}

	return nil
}

// MainTailBlock returns MainTailBlock.
func (bc *BlockChain) MainTailBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.mainTailBlock
}

// TailBlocks returns TailBlocks.
func (bc *BlockChain) TailBlocks() []*Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	blocks := make([]*Block, 0)
	for _, k := range bc.tailBlocks.Keys() {
		v, _ := bc.tailBlocks.Get(k)
		if v != nil {
			block := v.(*Block)
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// PutVerifiedNewBlocks put verified blocks and change tailBlocks
func (bc *BlockChain) PutVerifiedNewBlocks(parent *Block, allBlocks, tailBlocks []*Block) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	parentKey := parent.Hash().Str()
	if _, ok := bc.cachedBlocks.Get(parentKey); !ok {
		if _, err := bc.storage.Get(parent.Hash().Bytes()); err != nil {
			return ErrBlockNotExist
		}
	}
	for _, block := range allBlocks {
		bc.cachedBlocks.Add(block.Hash().Str(), block)
		if err := bc.storeBlock(block); err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("ailed to store the verified block")
			return err
		}

		// TODO handle metrics
	}

	if _, ok := bc.tailBlocks.Get(parentKey); ok {
		bc.tailBlocks.Remove(parentKey)
	}
	for _, block := range tailBlocks {
		addBlockToCache(block, bc.tailBlocks)
	}
	// TODO fork choice
	if bc.mainTailBlock.Hash() == parent.Hash() {
		bc.changeMainTailBlock(tailBlocks)
	}

	return nil
}

func (bc *BlockChain) changeMainTailBlock(tailBlocks []*Block) {
	bc.mainTailBlock = tailBlocks[0]
	maxHeight := bc.mainTailBlock.Height()
	for _, block := range tailBlocks[1:] {
		height := block.Height()
		if height > maxHeight {
			bc.mainTailBlock = block
			maxHeight = height
		}
	}
	if err := bc.storeBlockWithKey([]byte(tailBlockKeyInStorage), bc.mainTailBlock); err != nil {
		logging.Console().Error(err)
	}
}

func (bc *BlockChain) storeBlock(block *Block) error {
	return bc.storeBlockWithKey(block.Hash().Bytes(), block)
}

func (bc *BlockChain) storeBlockWithKey(key []byte, block *Block) error {
	pbBlock, err := block.ToProto()
	if err != nil {
		return err
	}
	value, err := proto.Marshal(pbBlock)
	if err != nil {
		return err
	}
	err = bc.storage.Put(key, value)
	if err != nil {
		return err
	}
	return nil
}

func addBlockToCache(block *Block, cache *lru.Cache) {
	cache.Add(block.Hash().Str(), block)
}
