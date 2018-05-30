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
	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

const (
	tailBlockKey = "blockchain_tail"
	libKey       = "blockchain_lib"
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

	consensus Consensus

	storage storage.Storage
}

// NewBlockChain return new BlockChain instance
func NewBlockChain(cfg *medletpb.Config) (*BlockChain, error) {
	bc := &BlockChain{
		chainID: cfg.Global.ChainId,
	}

	var err error
	bc.cachedBlocks, err = lru.New(int(cfg.Chain.BlockCacheSize))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"cacheSize": cfg.Chain.BlockCacheSize,
		}).Fatal("Failed to create block cache.")
		return nil, err
	}

	bc.tailBlocks, err = lru.New(int(cfg.Chain.TailCacheSize))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"cacheSize": cfg.Chain.TailCacheSize,
		}).Fatal("Failed to create tail block cache.")
		return nil, err
	}

	return bc, nil
}

// Setup sets up BlockChain.
func (bc *BlockChain) Setup(genesis *corepb.Genesis, consensus Consensus, stor storage.Storage) error {
	bc.genesis = genesis
	bc.storage = stor
	bc.consensus = consensus

	// Check if there is data in storage.
	_, err := bc.loadGenesisFromStorage()
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

	if !CheckGenesisConf(bc.genesisBlock, bc.genesis) {
		logging.Console().WithFields(logrus.Fields{
			"block":   bc.genesisBlock,
			"genesis": bc.genesis,
		}).Fatal("Failed to match genesis block and genesis configuration.")
		return ErrGenesisNotMatch
	}

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
		}).Fatal("Failed to load LIB from storage.")
		return err
	}
	return nil
}

// ChainID returns ChainID.
func (bc *BlockChain) ChainID() uint32 {
	return bc.chainID
}

// BlockByHash returns a block of given hash.
func (bc *BlockChain) BlockByHash(hash []byte) *Block {
	block, err := bc.loadBlockByHash(hash)
	if err != nil {
		return nil
	}
	return block
}

// BlockByHeight returns a block of given height.
func (bc *BlockChain) BlockByHeight(height uint64) *Block {
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
	ancestor, err := bc.FindCommonAncestorWithTail(newTail)
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

// FindCommonAncestorWithTail finds common ancestor with a current tail block.
func (bc *BlockChain) FindCommonAncestorWithTail(block *Block) (*Block, error) {
	tail := bc.mainTailBlock
	if tail.Height() > block.Height() {
		tail = bc.BlockByHeight(block.Height())
	}
	if tail == nil {
		return nil, ErrMissingParentBlock
	}

	return bc.findCommonAncestor(tail, block)
}

func (bc *BlockChain) findCommonAncestor(a, b *Block) (*Block, error) {
	if a.Height() > b.Height() {
		a = bc.BlockByHash(a.ParentHash())
		if a == nil {
			return nil, ErrMissingParentBlock
		}
	}

	for a.Height() < b.Height() {
		b = bc.BlockByHash(b.ParentHash())
		if b == nil {
			return nil, ErrMissingParentBlock
		}
	}

	for !byteutils.Equal(a.Hash(), b.Hash()) {
		a = bc.BlockByHash(a.ParentHash())
		b = bc.BlockByHash(b.ParentHash())
		if a == nil || b == nil {
			return nil, ErrMissingParentBlock
		}
	}
	return b, nil
}

func (bc *BlockChain) buildIndexByBlockHeight(from *Block, to *Block) error {
	for !byteutils.Equal(to.Hash(), from.Hash()) {
		err := bc.storage.Put(byteutils.FromUint64(to.height), to.Hash())
		if err != nil {
			logging.WithFields(logrus.Fields{
				"err":    err,
				"height": to.height,
				"hash":   byteutils.Bytes2Hex(to.Hash()),
			}).Error("Failed to put block index to storage.")
			return err
		}
		// TODO @cl9200 Remove tx in block from tx pool.
		to = bc.BlockByHash(to.ParentHash())
		if to == nil {
			logging.WithFields(logrus.Fields{
				"hash":       byteutils.Bytes2Hex(to.Hash()),
				"parentHash": byteutils.Bytes2Hex(to.ParentHash()),
			}).Error("Failed to find parent block when building block index by height.")
			return ErrMissingParentBlock
		}
	}
	return nil
}

func (bc *BlockChain) initGenesisToStorage() error {
	genesisBlock, err := NewGenesisBlock(bc.genesis, bc.consensus, bc.storage)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create new genesis block.")
		return err
	}
	if err = bc.storeBlock(genesisBlock); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to store new genesis block.")
		return err
	}
	if err = bc.storeTailHashToStorage(genesisBlock); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to update tail hash to new genesis block.")
		return err
	}
	if err = bc.storeHeightToStorage(genesisBlock); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to update block height of new genesis block. ")
		return err
	}
	if err = bc.storeLIBHashToStorage(genesisBlock); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to update lib to new genesis block.")
		return err
	}
	return nil
}

func (bc *BlockChain) loadBlockByHash(hash []byte) (*Block, error) {
	block := bc.loadBlockFromCache(hash)
	if block != nil {
		return block, nil
	}

	v, err := bc.storage.Get(hash)
	if err == storage.ErrKeyNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":  err,
			"hash": byteutils.Bytes2Hex(hash),
		}).Error("Failed to get block data from storage.")
		return nil, err
	}

	bd, err := bytesToBlockData(v)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to unmarshal block data.")
		return nil, err
	}

	block, err = bd.GetExecutedBlock(bc.consensus, bc.storage)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Error("Failed to get block from block data.")
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
	return bc.loadBlockByHash(hash)
}

func (bc *BlockChain) loadTailFromStorage() (*Block, error) {
	hash, err := bc.storage.Get([]byte(tailBlockKey))
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load tail block hash from storage.")
		return nil, err
	}
	return bc.loadBlockByHash(hash)
}

func (bc *BlockChain) loadLIBFromStorage() (*Block, error) {
	hash, err := bc.storage.Get([]byte(libKey))
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load lib hash from storage.")
		return nil, err
	}
	return bc.loadBlockByHash(hash)
}

func (bc *BlockChain) loadGenesisFromStorage() (*Block, error) {
	return bc.loadBlockByHeight(GenesisHeight)
}

func (bc *BlockChain) storeBlock(block *Block) error {
	pbBlock, err := block.ToProto()
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to convert block to proto.")
		return err
	}
	value, err := proto.Marshal(pbBlock)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
			"pb":    pbBlock,
		}).Error("Failed to marshal block.")
		return err
	}
	err = bc.storage.Put(block.Hash(), value)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to put block to storage.")
		return err
	}
	bc.cacheBlock(block)
	return nil
}

func (bc *BlockChain) storeTailHashToStorage(block *Block) error {
	return bc.storage.Put([]byte(tailBlockKey), block.Hash())
}

func (bc *BlockChain) storeLIBHashToStorage(block *Block) error {
	return bc.storage.Put([]byte(libKey), block.Hash())
}

func (bc *BlockChain) storeHeightToStorage(block *Block) error {
	height := byteutils.FromUint64(block.Height())
	return bc.storage.Put(height, block.Hash())
}

func (bc *BlockChain) cacheBlock(block *Block) {
	bc.cachedBlocks.Add(byteutils.Bytes2Hex(block.Hash()), block)
}

func (bc *BlockChain) loadBlockFromCache(hash []byte) *Block {
	v, ok := bc.cachedBlocks.Get(byteutils.Bytes2Hex(hash))
	if !ok {
		return nil
	}
	return v.(*Block)
}

func (bc *BlockChain) addToTailBlocks(block *Block) {
	bc.tailBlocks.Add(byteutils.Bytes2Hex(block.Hash()), block)
}

func (bc *BlockChain) removeFromTailBlocks(block *Block) {
	bc.tailBlocks.Remove(byteutils.Bytes2Hex(block.Hash()))
}
