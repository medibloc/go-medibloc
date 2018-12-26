package core

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/medibloc/go-medibloc/common"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// ChainManager manages blockchain features
type ChainManager struct {
	mu sync.RWMutex

	bc        *BlockChain
	tm        *TransactionManager
	consensus Consensus

	chainID       uint32
	mainTailBlock *Block
	lib           *Block

	tailBlocks *lru.Cache

	eventEmitter *EventEmitter

	TrigCh    chan bool
	quitCh    chan bool
	quitResCh chan bool
}

// NewChainManager returns new ChainManager instance
func NewChainManager(cfg *medletpb.Config, bc *BlockChain) (*ChainManager, error) {
	var err error

	cm := &ChainManager{
		bc:        bc,
		chainID:   cfg.Global.ChainId,
		TrigCh:    make(chan bool, 1),
		quitCh:    make(chan bool),
		quitResCh: make(chan bool),
	}
	cm.tailBlocks, err = lru.New(int(cfg.Chain.TailCacheSize))
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"cacheSize": cfg.Chain.TailCacheSize,
		}).Error("Failed to create tail block cache.")
		return nil, err
	}
	return cm, nil
}

// InjectEmitter injects event emitter
func (cm *ChainManager) InjectEmitter(emitter *EventEmitter) {
	cm.eventEmitter = emitter
}

// Setup setups chainManager
func (cm *ChainManager) Setup(tm *TransactionManager, consensus Consensus) error {
	cm.tm = tm
	cm.consensus = consensus

	// Set main tail block
	mainTailBlock, err := cm.bc.LoadTailFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load tail block from storage.")
		return err
	}
	cm.mainTailBlock = mainTailBlock
	cm.AddToTailBlocks(mainTailBlock)

	// Set LIB
	lib, err := cm.bc.LoadLIBFromStorage()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to load LIB from storage.")
		return err
	}

	cm.lib = lib

	return nil
}

// Start starts chainManager
func (cm *ChainManager) Start() {
	logging.Console().Info("Starting ChainManager...")
	go cm.loop()
}

// Stop stops chainManager
func (cm *ChainManager) Stop() {
	logging.Console().Info("Stopping ChainManager...")
	close(cm.quitCh)
	<-cm.quitResCh
}

func (cm *ChainManager) loop() {
	for {
		select {
		case <-cm.quitCh:
			close(cm.quitResCh)
			return
		case <-cm.TrigCh:
			cm.updateChain()
			continue
		}
	}
}

func (cm *ChainManager) updateChain() {
	newTail := cm.forkChoice()
	if byteutils.Equal(cm.mainTailBlock.Hash(), newTail.Hash()) {
		return
	}

	revertBlocks, newBlocks, err := cm.setTailBlock(newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set new tail block.")
		return
	}

	if err := cm.rearrangeTransactions(revertBlocks, newBlocks); err != nil {
		return
	}

	newLIB := cm.consensus.FindLIB(cm)
	if byteutils.Equal(cm.LIB().Hash(), newLIB.Hash()) {
		return
	}

	err = cm.SetLIB(newLIB)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set LIB.")
		return
	}

	logging.Console().WithFields(logrus.Fields{
		"LIB":         newLIB,
		"newMainTail": newTail,
	}).Info("Block accepted.")
	return
}

func (cm *ChainManager) rearrangeTransactions(revertBlock []*Block, newBlocks []*Block) error {
	addrNonce := make(map[common.Address]uint64)
	exclusiveFilter := make(map[*Transaction]bool)

	for _, newBlock := range newBlocks {
		for _, tx := range newBlock.Transactions() {
			if addrNonce[tx.From()] < tx.Nonce() {
				addrNonce[tx.From()] = tx.Nonce()
			}
			exclusiveFilter[tx] = true
		}
	}
	for addr, nonce := range addrNonce {
		cm.tm.DelByAddressNonce(addr, nonce)
	}

	// revert block
	pushTxs := make([]*Transaction, 0)
	for _, block := range revertBlock {
		for _, tx := range block.Transactions() {
			if _, ok := exclusiveFilter[tx]; !ok {
				pushTxs = append(pushTxs, tx)
			}
		}
		if cm.eventEmitter != nil {
			block.EmitBlockEvent(cm.eventEmitter, TopicRevertBlock)
		}
		logging.Console().Warn("A block is reverted.")
	}
	cm.tm.PushAndExclusiveBroadcast(pushTxs...)
	return nil
}

// ChainID return BlockChain.ChainID
func (cm *ChainManager) ChainID() uint32 {
	return cm.chainID
}

// MainTailBlock getter for mainTailBlock
func (cm *ChainManager) MainTailBlock() *Block {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.mainTailBlock
}

func (cm *ChainManager) setTailBlock(newTail *Block) ([]*Block, []*Block, error) {
	ancestor, err := cm.findAncestorOnCanonical(newTail, true)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":     err,
			"newTail": newTail,
		}).Error("Failed to find ancestor in canonical chain.")
		return nil, nil, err
	}
	mainTail := cm.MainTailBlock()

	// revert case
	blocks, err := cm.bc.LoadBetweenBlocks(ancestor, mainTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":  err,
			"from": ancestor,
			"to":   mainTail,
		}).Error("Failed to load blocks between ancestor and mainTail.")
		return nil, nil, err
	}

	newBlocks, err := cm.bc.BuildIndexByBlockHeight(ancestor, newTail)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":  err,
			"from": ancestor,
			"to":   newTail,
		}).Error("Failed to build index by block height.")
		return nil, nil, err
	}

	if err := cm.bc.StoreTailHashToStorage(newTail); err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"newTail": newTail,
		}).Error("Failed to store tail hash to storage.")
		return nil, nil, err
	}

	cm.mu.Lock()
	cm.mainTailBlock = newTail
	cm.mu.Unlock()

	if cm.eventEmitter != nil {
		newTail.EmitBlockEvent(cm.eventEmitter, TopicNewTailBlock)
	}
	return blocks, newBlocks, nil
}

// TailBlocks returns tailBlocks
func (cm *ChainManager) TailBlocks() []*Block {
	blocks := make([]*Block, 0, cm.tailBlocks.Len())
	for _, k := range cm.tailBlocks.Keys() {
		v, ok := cm.tailBlocks.Get(k)
		if ok {
			block := v.(*Block)
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// AddToTailBlocks adds blocks to tailBlocks
func (cm *ChainManager) AddToTailBlocks(block *Block) {
	cm.tailBlocks.Add(byteutils.Bytes2Hex(block.Hash()), block)
}

// RemoveFromTailBlocks removes block from tailBlocks
func (cm *ChainManager) RemoveFromTailBlocks(block *Block) {
	cm.tailBlocks.Remove(byteutils.Bytes2Hex(block.Hash()))
}

// LIB returns latest irreversible block of the chain.
func (cm *ChainManager) LIB() *Block {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.lib
}

// SetLIB sets LIB
func (cm *ChainManager) SetLIB(newLIB *Block) error {
	prevLIB := cm.LIB()
	if prevLIB.height > newLIB.height || byteutils.Equal(prevLIB.hash, newLIB.hash) {
		if cm.eventEmitter != nil {
			prevLIB.EmitBlockEvent(cm.eventEmitter, TopicLibBlock)
		}
		return nil
	}

	err := cm.bc.StoreLIBHashToStorage(newLIB)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":     err,
			"new LIB": newLIB,
		}).Error("Failed to store LIB hash to storage.")
		return err
	}

	cm.mu.Lock()
	cm.lib = newLIB
	cm.mu.Unlock()

	if cm.eventEmitter != nil {
		newLIB.EmitBlockEvent(cm.eventEmitter, TopicLibBlock)
	}

	for _, tail := range cm.TailBlocks() {
		if !cm.IsForkedBeforeLIB(tail) {
			continue
		}
		if err := cm.removeForkedBranch(tail); err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":   err,
				"block": tail,
			}).Error("Failed to remove a forked branch.")
			return err
		}
	}
	return nil
}

func (cm *ChainManager) findAncestorOnCanonical(block *Block, breakAtLIB bool) (*Block, error) {
	var err error

	cm.mu.RLock()
	tail := cm.mainTailBlock
	cm.mu.RUnlock()

	for tail.Height() < block.Height() {
		block, err = cm.bc.ParentBlock(block)
		if err != nil {
			return nil, err
		}
	}

	lib := cm.LIB()
	for {
		if breakAtLIB && block.height < lib.Height() {
			return nil, ErrMissingParentBlock
		}

		canonical := cm.bc.BlockByHeight(block.Height())
		if canonical == nil {
			logging.Console().WithFields(logrus.Fields{
				"height": block.Height(),
			}).Error("Failed to get the block of height.")
			return nil, ErrMissingParentBlock
		}

		if byteutils.Equal(canonical.Hash(), block.Hash()) {
			break
		}

		block, err = cm.bc.ParentBlock(block)
		if err != nil {
			return nil, err
		}

	}
	return block, nil
}

// IsForkedBeforeLIB checks that if the block is forked before LIB
func (cm *ChainManager) IsForkedBeforeLIB(block *Block) bool {
	_, err := cm.findAncestorOnCanonical(block, true)
	if err != nil {
		return true
	}
	return false
}

func (cm *ChainManager) removeForkedBranch(tail *Block) error {
	ancestor, err := cm.findAncestorOnCanonical(tail, false)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to find ancestor in canonical.")
		return err
	}

	block := tail
	for !byteutils.Equal(ancestor.Hash(), block.Hash()) {
		err = cm.bc.RemoveBlock(block)
		if err != nil {
			return err
		}

		block, err = cm.bc.ParentBlock(block)
		if err != nil {
			return err
		}
	}

	cm.RemoveFromTailBlocks(tail)
	return nil
}

// BlockByHeight returns block by height
func (cm *ChainManager) BlockByHeight(height uint64) *Block {
	if height > cm.MainTailBlock().Height() {
		return nil
	}
	return cm.bc.BlockByHeight(height)
}

// BlockByHash returns block by hash
func (cm *ChainManager) BlockByHash(hash []byte) *Block {
	return cm.bc.BlockByHash(hash)
}

func (cm *ChainManager) forkChoice() *Block {
	newTail := cm.MainTailBlock()
	tails := cm.TailBlocks()
	for _, block := range tails {
		if cm.IsForkedBeforeLIB(block) {
			logging.WithFields(logrus.Fields{
				"block": block,
				"lib":   cm.LIB(),
			}).Debug("Blocks forked before LIB can not be selected.")
			continue
		}
		if block.Height() > newTail.Height() {
			newTail = block
		}
	}

	return newTail
}
