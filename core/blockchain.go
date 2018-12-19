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

	corepb "github.com/medibloc/go-medibloc/core/pb"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"

	"github.com/gogo/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
)

const (
	blockByHashPrefix   = "bhash_"
	blockByHeightPrefix = "bheight_"
	tailBlockKey        = "blockchain_tail"
	libKey              = "blockchain_lib"
)

// BlockChain manages blockchain structure.
type BlockChain struct {
	mu sync.RWMutex

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

	eventEmitter *EventEmitter
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
		}).Error("Failed to create block cache.")
		return nil, err
	}

	bc.tailBlocks, err = lru.New(int(cfg.Chain.TailCacheSize))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"cacheSize": cfg.Chain.TailCacheSize,
		}).Error("Failed to create tail block cache.")
		return nil, err
	}

	return bc, nil
}

// SetEventEmitter set emitter to blockchian
func (bc *BlockChain) SetEventEmitter(emitter *EventEmitter) {
	bc.eventEmitter = emitter
}

// Setup sets up BlockChain.
func (bc *BlockChain) Setup(genesis *corepb.Genesis, consensus Consensus, stor storage.Storage) error {
	bc.genesis = genesis
	bc.storage = stor
	bc.consensus = consensus

	// Load Genesis Block
	genesisBlock, err := bc.loadGenesisFromStorage()
	if err != nil && err != storage.ErrKeyNotFound {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load genesis block from storage.")
		return err
	}
	if genesisBlock == nil || err == storage.ErrKeyNotFound {
		err = bc.initGenesisToStorage()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to initialize genesis block to storage.")
			return err
		}
		genesisBlock, err = bc.loadGenesisFromStorage()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to load genesis block from storage.")
			return err
		}
	}

	if !CheckGenesisConf(genesisBlock, bc.genesis) {
		logging.Console().WithFields(logrus.Fields{
			"block":   genesisBlock,
			"genesis": bc.genesis,
		}).Error("Failed to match genesis block and genesis configuration.")
		return ErrGenesisNotMatch
	}

	// Set Genesis Block
	bc.genesisBlock = genesisBlock

	// Load Tail Block
	mainTailBlock, err := bc.loadTailFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load tail block from storage.")
		return err
	}
	// Set TailBlock
	bc.mainTailBlock = mainTailBlock
	bc.addToTailBlocks(bc.mainTailBlock)

	// Load LIB
	lib, err := bc.loadLIBFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load LIB from storage.")
		return err
	}

	// Set LIB
	bc.lib = lib

	// Reindexing Blockchain from mainTailBlock to lib from storage
	if mainTailBlock == nil || mainTailBlock.Height() == 1 {
		return nil
	}
	_, err = bc.buildIndexByBlockHeight(lib, mainTailBlock)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to reindexing blockchain.")
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
func (bc *BlockChain) BlockByHeight(height uint64) (*Block, error) {
	bc.mu.RLock()
	if height > bc.mainTailBlock.Height() {
		bc.mu.RUnlock()
		return nil, ErrBlockNotExist
	}
	bc.mu.RUnlock()

	block, err := bc.loadBlockByHeight(height)
	if err != nil {
		return nil, ErrBlockNotExist
	}
	return block, nil
}

// MainTailBlock returns MainTailBlock.
func (bc *BlockChain) MainTailBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.mainTailBlock
}

// LIB returns latest irreversible block.
func (bc *BlockChain) LIB() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
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

// PutVerifiedNewBlock put verified block on the chain
func (bc *BlockChain) PutVerifiedNewBlock(parent, child *Block) error {
	if bc.BlockByHash(parent.Hash()) == nil {
		logging.WithFields(logrus.Fields{
			"block": parent,
		}).Error("Failed to find parent block.")
		return ErrBlockNotExist
	}

	if err := bc.storeBlock(child); err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": child,
		}).Error("Failed to store the verified block")
		return err
	}
	logging.WithFields(logrus.Fields{
		"block": child,
	}).Info("Accepted the new block on chain")
	bc.addToTailBlocks(child)
	bc.removeFromTailBlocks(parent)

	if bc.eventEmitter != nil {
		child.EmitTxExecutionEvent(bc.eventEmitter)
		child.EmitBlockEvent(bc.eventEmitter, TopicAcceptedBlock)
	}

	return nil
}

// SetLIB sets LIB.
func (bc *BlockChain) SetLIB(newLIB *Block) error {
	bc.mu.RLock()
	prevLIB := bc.lib
	bc.mu.RUnlock()
	if prevLIB.height > newLIB.height || byteutils.Equal(prevLIB.hash, newLIB.hash) {
		if bc.eventEmitter != nil {
			event := &Event{
				Topic: TopicLibBlock,
				Data:  byteutils.Bytes2Hex(prevLIB.Hash()),
				Type:  "",
			}
			bc.eventEmitter.Trigger(event)
		}
		return nil
	}

	err := bc.storeLIBHashToStorage(newLIB)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"newTail": newLIB,
		}).Error("Failed to store LIB hash to storage.")
		return err
	}

	bc.mu.Lock()
	bc.lib = newLIB
	bc.mu.Unlock()

	if bc.eventEmitter != nil {
		newLIB.EmitBlockEvent(bc.eventEmitter, TopicLibBlock)
	}

	for _, tail := range bc.TailBlocks() {
		if !bc.IsForkedBeforeLIB(tail) {
			continue
		}
		err = bc.removeForkedBranch(tail)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":   err,
				"block": tail,
			}).Error("Failed to remove a forked branch.")
		}
	}
	return nil
}

// SetTailBlock sets tail block.
func (bc *BlockChain) SetTailBlock(newTail *Block) ([]*Block, []*Block, error) {
	ancestor, err := bc.FindAncestorOnCanonical(newTail, true)
	bc.mu.RLock()
	mainTail := bc.mainTailBlock
	bc.mu.RUnlock()

	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":     err,
			"newTail": newTail,
			"tail":    mainTail,
		}).Error("Failed to find ancestor in canonical chain.")
		return nil, nil, err
	}

	// revert case
	blocks, err := bc.loadBetweenBlocks(ancestor, mainTail)
	if err != nil {
		return nil, nil, err
	}

	bc.mu.Lock()
	newBlocks, err := bc.buildIndexByBlockHeight(ancestor, newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":  err,
			"from": ancestor,
			"to":   newTail,
		}).Error("Failed to build index by block height.")
		bc.mu.Unlock()
		return nil, nil, err
	}
	bc.mu.Unlock()

	if err = bc.storeTailHashToStorage(newTail); err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"newTail": newTail,
		}).Error("Failed to store tail hash to storage.")
		return nil, nil, err
	}

	bc.mu.Lock()
	bc.mainTailBlock = newTail
	bc.mu.Unlock()

	if bc.eventEmitter != nil {
		newTail.EmitBlockEvent(bc.eventEmitter, TopicNewTailBlock)
	}

	return blocks, newBlocks, nil
}

// FindAncestorOnCanonical finds most recent ancestor block in canonical chain.
func (bc *BlockChain) FindAncestorOnCanonical(block *Block, breakAtLIB bool) (*Block, error) {
	var err error

	bc.mu.RLock()
	tail := bc.mainTailBlock
	bc.mu.RUnlock()

	for tail.Height() < block.Height() {
		block, err = bc.parentBlock(block)
		if err != nil {
			return nil, err
		}
	}

	lib := bc.LIB()
	for {
		if breakAtLIB && block.height < lib.Height() {
			return nil, ErrMissingParentBlock
		}

		canonical, err := bc.BlockByHeight(block.Height())
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"height": block.Height(),
			}).Error("Failed to get the block of height.")
			return nil, ErrMissingParentBlock
		}

		if byteutils.Equal(canonical.Hash(), block.Hash()) {
			break
		}

		block, err = bc.parentBlock(block)
		if err != nil {
			return nil, err
		}

	}
	return block, nil
}

// IsForkedBeforeLIB checks if the block is forked before LIB.
func (bc *BlockChain) IsForkedBeforeLIB(block *Block) bool {
	_, err := bc.FindAncestorOnCanonical(block, true)
	if err != nil {
		return true
	}
	return false
}

func (bc *BlockChain) parentBlock(block *Block) (*Block, error) {
	parentHash := block.ParentHash()
	block = bc.BlockByHash(parentHash)
	if block == nil {
		logging.Console().WithFields(logrus.Fields{
			"hash": parentHash,
		}).Error("Failed to find a parent block by hash.")
		return nil, ErrMissingParentBlock
	}
	return block, nil
}

func (bc *BlockChain) loadBetweenBlocks(from *Block, to *Block) ([]*Block, error) {
	// Normal case (ancestor === mainTail)
	if byteutils.Equal(from.Hash(), to.Hash()) {
		return nil, nil
	}

	var blocks []*Block
	// Revert case
	revertBlock := to
	var err error

	for !byteutils.Equal(from.Hash(), revertBlock.Hash()) {
		blocks = append(blocks, revertBlock)
		revertBlock, err = bc.parentBlock(revertBlock)
		if err != nil {
			return nil, err
		}
	}
	return blocks, nil
}

func (bc *BlockChain) buildIndexByBlockHeight(from *Block, to *Block) ([]*Block, error) {
	var blocks []*Block

	for !byteutils.Equal(to.Hash(), from.Hash()) {
		err := bc.storeHeightToStorage(to)
		if err != nil {
			logging.WithFields(logrus.Fields{
				"err":    err,
				"height": to.height,
				"hash":   byteutils.Bytes2Hex(to.Hash()),
			}).Error("Failed to put block index to storage.")
			return nil, err
		}
		blocks = append(blocks, to)

		to, err = bc.parentBlock(to)
		if err != nil {
			return nil, err
		}
	}
	return blocks, nil
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

	v, err := bc.storage.Get(append([]byte(blockByHashPrefix), hash...))
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

	bd, err := BytesToBlockData(v)
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
	hash, err := bc.storage.Get(append([]byte(blockByHeightPrefix), byteutils.FromUint64(height)...))
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
	err = bc.storage.Put(append([]byte(blockByHashPrefix), block.Hash()...), value)
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
	return bc.storage.Put(append([]byte(blockByHeightPrefix), height...), block.Hash())
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

func (bc *BlockChain) removeBlock(block *Block) error {
	canonical, err := bc.BlockByHeight(block.Height())

	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"height": block.Height(),
		}).Error("Failed to get the block of height.")
		return ErrCannotRemoveBlockOnCanonical
	}

	if canonical != nil && byteutils.Equal(canonical.Hash(), block.Hash()) {
		logging.Console().WithFields(logrus.Fields{
			"block": block,
		}).Error("Can not remove block on canonical chain.")
		return ErrCannotRemoveBlockOnCanonical
	}

	err = bc.storage.Delete(block.Hash())
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to delete blocks in storage.")
		return err
	}
	bc.cachedBlocks.Remove(byteutils.Bytes2Hex(block.Hash()))
	return nil
}

func (bc *BlockChain) removeForkedBranch(tail *Block) error {
	ancestor, err := bc.FindAncestorOnCanonical(tail, false)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to find ancestor in canonical.")
		return err
	}

	block := tail
	for !byteutils.Equal(ancestor.Hash(), block.Hash()) {
		err = bc.removeBlock(block)
		if err != nil {
			return err
		}

		block, err = bc.parentBlock(block)
		if err != nil {
			return err
		}
	}

	bc.removeFromTailBlocks(tail)
	return nil
}
