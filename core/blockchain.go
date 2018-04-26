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
	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

const (
	blockCacheSize = 128
	tailCacheSize  = 128
	tailBlockKey   = "blockchain_tail"
	libKey         = "blockchain_lib"
)

// BlockChain manages blockchain structure.
type BlockChain struct {
	chainID uint32

	genesis *corepb.Genesis

	genesisBlock  *Block
	mainTailBlock *Block
	lib           *Block

	cachedBlocks *lru.Cache
	// tailBlocks all tail blocks including mainTailBlock
	tailBlocks *lru.Cache

	storage storage.Storage
}

// NewBlockChain return new BlockChain instance
func NewBlockChain(cfg *medletpb.Config) (*BlockChain, error) {
	bc := &BlockChain{
		chainID: cfg.Chain.ChainId,
	}

	var err error
	bc.cachedBlocks, err = lru.New(blockCacheSize)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"cacheSize": blockCacheSize,
		}).Fatal("Failed to create block cache.")
		return nil, err
	}

	bc.tailBlocks, err = lru.New(tailCacheSize)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"cacheSize": tailCacheSize,
		}).Fatal("Failed to create tail block cache.")
		return nil, err
	}

	return bc, nil
}

// Setup sets up BlockChain.
func (bc *BlockChain) Setup(genesis *corepb.Genesis, stor storage.Storage) error {
	bc.genesis = genesis
	bc.storage = stor

	// Check if there is data in storage.
	_, err := bc.loadTailFromStorage()
	if err != nil && err != storage.ErrKeyNotFound {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to load tail block from storage.")
		return err
	}
	if err != nil && err == storage.ErrKeyNotFound {
		err = bc.initGenesisToStorage()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Fatal("Failed to initialize genesis block to storage.")
			return err
		}
	}

	// Load genesis block
	bc.genesisBlock, err = bc.loadGenesisFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to load genesis block from storage.")
		return err
	}

	// TODO @cl9200 Make sure that bc.genesis and bc.genesisBlock are the same genesis settings.
	// checkGenesisConfig(bc.genesisBlock, bc.genesis)

	// Load tail block
	bc.mainTailBlock, err = bc.loadTailFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to load tail block from storage.")
		return err
	}
	bc.addToTailBlocks(bc.mainTailBlock)

	// Load LIB
	bc.lib, err = bc.loadLIBFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Fatal("Failed to LIB from storage.")
		return err
	}
	return nil
}

// ChainID returns ChainID.
func (bc *BlockChain) ChainID() uint32 {
	return bc.chainID
}

// BlockByHash returns a block of given hash.
func (bc *BlockChain) BlockByHash(hash common.Hash) *Block {
	block, err := bc.loadBlockByHash(hash)
	if err != nil {
		return nil
	}
	return block
}

// BlockOnCanonicalByHash returns a block of given hash.
func (bc *BlockChain) BlockOnCanonicalByHash(hash common.Hash) *Block {
	blockByHash, err := bc.loadBlockByHash(hash)
	if err != nil {
		return nil
	}
	blockByHeight := bc.BlockOnCanonicalByHeight(blockByHash.Height())
	if blockByHeight == nil {
		return nil
	}
	if !blockByHeight.Hash().Equals(blockByHash.Hash()) {
		return nil
	}
	return blockByHeight
}

// BlockOnCanonicalByHeight returns a block of given height.
func (bc *BlockChain) BlockOnCanonicalByHeight(height uint64) *Block {
	if height > bc.mainTailBlock.Height() {
		return nil
	}
	block, err := bc.loadBlockByHeight(height)
	if err != nil {
		return nil
	}
	return block
}

// MainTailBlock returns MainTailBlock.
func (bc *BlockChain) MainTailBlock() *Block {
	return bc.mainTailBlock
}

// LIB returns latest irreversible block.
func (bc *BlockChain) LIB() *Block {
	return bc.lib
}

// TailBlocks returns TailBlocks.
func (bc *BlockChain) TailBlocks() []*Block {
	blocks := make([]*Block, 0, bc.tailBlocks.Len())
	for _, k := range bc.tailBlocks.Keys() {
		v, ok := bc.tailBlocks.Get(k)
		if ok {
			block := v.(*Block)
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// PutVerifiedNewBlocks put verified blocks and change tailBlocks
func (bc *BlockChain) PutVerifiedNewBlocks(parent *Block, allBlocks, tailBlocks []*Block) error {
	if bc.BlockByHash(parent.Hash()) == nil {
		logging.WithFields(logrus.Fields{
			"block": parent,
		}).Error("Failed to find parent block.")
		return ErrBlockNotExist
	}

	for _, block := range allBlocks {
		if err := bc.storeBlock(block); err != nil {
			logging.WithFields(logrus.Fields{
				"err":   err,
				"block": block,
			}).Error("Failed to store the verified block")
			return err
		}
		logging.WithFields(logrus.Fields{
			"block": block,
		}).Info("Accepted the new block on chain")
	}

	for _, block := range tailBlocks {
		bc.addToTailBlocks(block)
	}
	bc.removeFromTailBlocks(parent)

	return nil
}

// SetLIB sets LIB.
func (bc *BlockChain) SetLIB(newLIB *Block) error {
	err := bc.storeLIBHashToStorage(newLIB)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"newTail": newLIB,
		}).Error("Failed to store LIB hash to storage.")
		return err
	}
	bc.lib = newLIB
	return nil
}

// SetTailBlock sets tail block.
func (bc *BlockChain) SetTailBlock(newTail *Block) error {
	ancestor, err := bc.findCommonAncestorWithTail(newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"newTail": newTail,
			"tail":    bc.mainTailBlock,
		}).Error("Failed to find common ancestor with tail.")
		return err
	}

	// TODO @cl9200 revertBlocks

	if err = bc.buildIndexByBlockHeight(ancestor, newTail); err != nil {
		logging.WithFields(logrus.Fields{
			"err":  err,
			"from": ancestor,
			"to":   newTail,
		}).Error("Failed to build index by block height.")
		return err
	}

	if err = bc.storeTailHashToStorage(newTail); err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"newTail": newTail,
		}).Error("Failed to store tail hash to storage.")
		return err
	}
	bc.mainTailBlock = newTail
	return nil
}

func (bc *BlockChain) buildIndexByBlockHeight(from *Block, to *Block) error {
	for !to.Hash().Equals(from.Hash()) {
		err := bc.storage.Put(byteutils.FromUint64(to.height), to.Hash().Bytes())
		if err != nil {
			return err
		}
		// TODO @cl9200 Remove tx in block from tx pool.
		to = bc.BlockByHash(to.ParentHash())
		if to == nil {
			return ErrMissingParentBlock
		}
	}
	return nil
}

func (bc *BlockChain) findCommonAncestorWithTail(block *Block) (*Block, error) {
	tail := bc.mainTailBlock
	if tail.Height() > block.Height() {
		tail = bc.BlockOnCanonicalByHeight(block.Height())
	}
	if tail == nil {
		return nil, ErrMissingParentBlock
	}

	for tail.Height() < block.Height() {
		block = bc.BlockByHash(block.ParentHash())
		if block == nil {
			return nil, ErrMissingParentBlock
		}
	}

	for !tail.Hash().Equals(block.Hash()) {
		tail = bc.BlockByHash(tail.ParentHash())
		block = bc.BlockByHash(block.ParentHash())
		if tail == nil || block == nil {
			return nil, ErrMissingParentBlock
		}
	}
	return block, nil
}

func (bc *BlockChain) initGenesisToStorage() error {
	genesisBlock, err := NewGenesisBlock(bc.genesis, bc.storage)
	if err != nil {
		return err
	}
	if err = bc.storeBlock(genesisBlock); err != nil {
		return err
	}
	if err = bc.storeTailHashToStorage(genesisBlock); err != nil {
		return err
	}
	if err = bc.storeHeightToStorage(genesisBlock); err != nil {
		return err
	}
	return bc.storeLIBHashToStorage(genesisBlock)
}

func (bc *BlockChain) loadBlockByHash(hash common.Hash) (*Block, error) {
	block := bc.loadBlockFromCache(hash)
	if block != nil {
		return block, nil
	}
	v, err := bc.storage.Get(hash.Bytes())
	if err != nil {
		return nil, err
	}
	bd, err := bytesToBlockData(v)
	if err != nil {
		return nil, err
	}
	block, err = bd.GetExecutedBlock(bc.storage)
	if err != nil {
		return nil, err
	}
	bc.cacheBlock(block)
	return block, nil
}

func (bc *BlockChain) loadBlockByHeight(height uint64) (*Block, error) {
	hash, err := bc.storage.Get(byteutils.FromUint64(height))
	if err != nil {
		return nil, err
	}
	return bc.loadBlockByHash(common.BytesToHash(hash))
}

func (bc *BlockChain) loadTailFromStorage() (*Block, error) {
	hash, err := bc.storage.Get([]byte(tailBlockKey))
	if err != nil {
		return nil, err
	}
	return bc.loadBlockByHash(common.BytesToHash(hash))
}

func (bc *BlockChain) loadLIBFromStorage() (*Block, error) {
	hash, err := bc.storage.Get([]byte(libKey))
	if err != nil {
		return nil, err
	}
	return bc.loadBlockByHash(common.BytesToHash(hash))
}

func (bc *BlockChain) loadGenesisFromStorage() (*Block, error) {
	return bc.loadBlockByHeight(GenesisHeight)
}

func (bc *BlockChain) storeBlock(block *Block) error {
	pbBlock, err := block.ToProto()
	if err != nil {
		return err
	}
	value, err := proto.Marshal(pbBlock)
	if err != nil {
		return err
	}
	err = bc.storage.Put(block.Hash().Bytes(), value)
	if err != nil {
		return err
	}
	bc.cacheBlock(block)
	return nil
}

func (bc *BlockChain) storeTailHashToStorage(block *Block) error {
	return bc.storage.Put([]byte(tailBlockKey), block.Hash().Bytes())
}

func (bc *BlockChain) storeLIBHashToStorage(block *Block) error {
	return bc.storage.Put([]byte(libKey), block.Hash().Bytes())
}

func (bc *BlockChain) storeHeightToStorage(block *Block) error {
	height := byteutils.FromUint64(block.Height())
	return bc.storage.Put(height, block.Hash().Bytes())
}

func (bc *BlockChain) cacheBlock(block *Block) {
	bc.cachedBlocks.Add(block.Hash().Str(), block)
}

func (bc *BlockChain) loadBlockFromCache(hash common.Hash) *Block {
	v, ok := bc.cachedBlocks.Get(hash.Str())
	if !ok {
		return nil
	}
	return v.(*Block)
}

func (bc *BlockChain) addToTailBlocks(block *Block) {
	bc.tailBlocks.Add(block.Hash().Str(), block)
}

func (bc *BlockChain) removeFromTailBlocks(block *Block) {
	bc.tailBlocks.Remove(block.Hash().Str())
}
