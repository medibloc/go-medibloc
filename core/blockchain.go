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
	tailBlockKey = "blockchain_tail"
	libKey       = "blockchain_lib"
)

// BlockChain manages blockchain structure.
type BlockChain struct {
	mu sync.RWMutex

	genesis *corepb.Genesis

	genesisBlock *Block

	cachedBlocks *lru.Cache

	consensus Consensus

	storage storage.Storage

	eventEmitter *EventEmitter
}

// NewBlockChain return new BlockChain instance
func NewBlockChain(cfg *medletpb.Config) (*BlockChain, error) {
	bc := &BlockChain{}

	var err error
	bc.cachedBlocks, err = lru.New(int(cfg.Chain.BlockCacheSize))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"cacheSize": cfg.Chain.BlockCacheSize,
		}).Error("Failed to create block cache.")
		return nil, err
	}

	return bc, nil
}

// SetEventEmitter set emitter to blockchian
func (bc *BlockChain) SetEventEmitter(emitter *EventEmitter) {
	bc.eventEmitter = emitter
}

// Setup sets up BlockChain.
func (bc *BlockChain) Setup(genesis *corepb.Genesis, consensus Consensus, txMap TxFactory, stor storage.Storage) error {
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
		err = bc.initGenesisToStorage(txMap)
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

	mainTailBlock, err := bc.LoadTailFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load tail block from storage.")
		return err
	}
	lib, err := bc.LoadLIBFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load LIB from storage.")
		return err
	}

	// Reindexing Blockchain from mainTailBlock to lib from storage
	if mainTailBlock == nil || mainTailBlock.Height() == 1 {
		return nil
	}
	_, err = bc.BuildIndexByBlockHeight(lib, mainTailBlock)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to reindexing blockchain.")
		return err
	}

	return nil
}

func (bc *BlockChain) initGenesisToStorage(txMap TxFactory) error {
	genesisBlock, err := NewGenesisBlock(bc.genesis, bc.consensus, txMap, bc.storage)
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
	if err = bc.StoreTailHashToStorage(genesisBlock); err != nil {
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
	if err = bc.StoreLIBHashToStorage(genesisBlock); err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to update lib to new genesis block.")
		return err
	}
	return nil
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
	block, err := bc.loadBlockByHeight(height)
	if err != nil {
		return nil
	}
	return block
}

// ParentBlock returns parent block
func (bc *BlockChain) ParentBlock(block *Block) (*Block, error) {
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

// LoadBetweenBlocks returns blocks between fromBlock and toBlock
func (bc *BlockChain) LoadBetweenBlocks(from *Block, to *Block) ([]*Block, error) {
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
		revertBlock, err = bc.ParentBlock(revertBlock)
		if err != nil {
			return nil, err
		}
	}
	return blocks, nil
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

	if bc.eventEmitter != nil {
		child.EmitTxExecutionEvent(bc.eventEmitter)
		child.EmitBlockEvent(bc.eventEmitter, TopicAcceptedBlock)
	}

	return nil
}

// BuildIndexByBlockHeight save block with height key
func (bc *BlockChain) BuildIndexByBlockHeight(from *Block, to *Block) ([]*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	var blocks []*Block

	for !byteutils.Equal(to.Hash(), from.Hash()) {
		err := bc.storage.Put(byteutils.FromUint64(to.height), to.Hash())
		if err != nil {
			logging.WithFields(logrus.Fields{
				"err":    err,
				"height": to.height,
				"hash":   byteutils.Bytes2Hex(to.Hash()),
			}).Error("Failed to put block index to storage.")
			return nil, err
		}
		blocks = append(blocks, to)

		to, err = bc.ParentBlock(to)
		if err != nil {
			return nil, err
		}
	}
	return blocks, nil
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
	hash, err := bc.storage.Get(byteutils.FromUint64(height))
	if err != nil {
		return nil, err
	}
	return bc.loadBlockByHash(hash)
}

// LoadTailFromStorage loads tail block from storage
func (bc *BlockChain) LoadTailFromStorage() (*Block, error) {
	hash, err := bc.storage.Get([]byte(tailBlockKey))
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load tail block hash from storage.")
		return nil, err
	}
	return bc.loadBlockByHash(hash)
}

// LoadLIBFromStorage loads LIB from storage
func (bc *BlockChain) LoadLIBFromStorage() (*Block, error) {
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

// StoreTailHashToStorage stores tail hash to storage
func (bc *BlockChain) StoreTailHashToStorage(block *Block) error {
	return bc.storage.Put([]byte(tailBlockKey), block.Hash())
}

// StoreLIBHashToStorage stores LIB hash to storage
func (bc *BlockChain) StoreLIBHashToStorage(block *Block) error {
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

// RemoveBlock removes block from stroage
func (bc *BlockChain) RemoveBlock(block *Block) error {
	canonical := bc.BlockByHeight(block.Height())

	if canonical == nil {
		logging.Console().WithFields(logrus.Fields{
			"height": block.Height(),
		}).Error("Failed to get the block of height.")
		return ErrCannotRemoveBlockOnCanonical
	}

	if byteutils.Equal(canonical.Hash(), block.Hash()) {
		logging.Console().WithFields(logrus.Fields{
			"block": block,
		}).Error("Can not remove block on canonical chain.")
		return ErrCannotRemoveBlockOnCanonical
	}

	err := bc.storage.Delete(block.Hash())
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
