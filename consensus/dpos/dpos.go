package dpos

import (
	"time"

	"errors"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Consensus properties.
const (
	BlockInterval   = 15 * time.Second
	DynasyInterval  = 210 * BlockInterval
	DynastySize     = 21
	ConsensusSize   = DynastySize*2/3 + 1
	MinMintDuration = 2 * time.Second

	miningTickInterval = time.Second
)

// Consensus error types.
var (
	ErrInvalidBlockInterval   = errors.New("invalid block interval")
	ErrInvalidBlockProposer   = errors.New("invalid block proposer")
	ErrInvalidBlockForgeTime  = errors.New("invalid time to forge block")
	ErrFoundNilProposer       = errors.New("found a nil proposer")
	ErrBlockMintedInNextSlot  = errors.New("cannot mint block now, there is a block minted in current slot")
	ErrWaitingBlockInLastSlot = errors.New("cannot mint block now, waiting for last block")
)

// Dpos returns dpos consensus model.
type Dpos struct {
	coinbase common.Address
	miner    common.Address

	bm *core.BlockManager
	tm *core.TransactionManager

	quitCh chan int
}

// New returns dpos consensus.
func New(cfg *medletpb.Config) *Dpos {
	return &Dpos{
		coinbase: common.HexToAddress(cfg.Chain.Coinbase),
		miner:    common.HexToAddress(cfg.Chain.Miner),
		quitCh:   make(chan int, 1),
	}
}

// Setup sets up dpos.
func (d *Dpos) Setup(bm *core.BlockManager, tm *core.TransactionManager) {
	d.bm = bm
	d.tm = tm
}

// Start starts miner.
func (d *Dpos) Start() {
	go d.loop()
}

// Stop stops miner.
func (d *Dpos) Stop() {
	d.quitCh <- 0
}

// ForkChoice chooses fork.
func (d *Dpos) ForkChoice(bc *core.BlockChain) (newTail *core.Block) {
	// TODO @cl9200 Filter tails that are forked before LIB.
	tail := bc.MainTailBlock()
	tails := bc.TailBlocks()
	for _, block := range tails {
		if block.Height() > tail.Height() {
			tail = block
		}
	}

	return tail
}

// FindLIB finds new LIB.
func (d *Dpos) FindLIB(bc *core.BlockChain) (newLIB *core.Block) {
	lib := bc.LIB()
	tail := bc.MainTailBlock()
	cur := tail
	confirmed := make(map[string]bool)
	members, err := tail.State().Dynasty()
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":  err,
			"tail": tail,
			"lib":  lib,
		}).Error("Failed to get members of dynasty.")
		return
	}
	dynastyGen := int64(-1)

	for !cur.Hash().Equals(lib.Hash()) {
		if dynastyGen != dynastyGenByTime(cur.Timestamp()) {
			dynastyGen = dynastyGenByTime(cur.Timestamp())
			confirmed = make(map[string]bool)
			// TODO @cl9200 Replace member of dynasty.
			// members, err = tail.State().Dynasty()
		}

		if cur.Height()-lib.Height() < uint64(ConsensusSize-len(confirmed)) {
			return
		}

		proposer, err := findProposer(cur.Timestamp(), members)
		if err != nil {
			return
		}

		confirmed[proposer.Hex()] = true
		if len(confirmed) >= ConsensusSize {
			return cur
		}

		cur = bc.BlockByHash(cur.ParentHash())
		if cur == nil {
			logging.WithFields(logrus.Fields{
				"tail": tail,
				"lib":  lib,
			}).Error("Failed to find LIB.")
		}
	}
	return lib
}

func dynastyGenByTime(ts int64) int64 {
	now := time.Duration(ts) * time.Second
	return int64(now / DynasyInterval)
}

// VerifyProposer verifies block proposer.
func (d *Dpos) VerifyProposer(block *core.BlockData) error {
	// TODO @cl9200 Handling when tail height is higher than block height.
	tail := d.bm.TailBlock()
	elapsed := time.Duration(block.Timestamp()-tail.Timestamp()) * time.Second
	if elapsed%BlockInterval != 0 {
		return ErrInvalidBlockInterval
	}

	members, err := tail.State().Dynasty()
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":   err,
			"block": tail,
		}).Debug("Failed to get members of dynasty.")
		return err
	}

	proposer, err := findProposer(block.Timestamp(), members)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"blockTime": block.Timestamp(),
			"members":   members,
		}).Debug("Failed to find block proposer.")
		return err
	}

	err = verifyBlockSign(proposer, block)
	if err != nil {
		return err
	}

	return nil
}

func verifyBlockSign(proposer *common.Address, block *core.BlockData) error {
	signer, err := recoverSignerFromSignature(block.Alg(), block.Hash().Bytes(), block.Signature())
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":      err,
			"proposer": proposer,
			"signer":   signer,
			"block":    block,
		}).Error("Failed to recover block's signer.")
		return err
	}
	if !proposer.Equals(signer) {
		logging.WithFields(logrus.Fields{
			"signer":   signer,
			"proposer": proposer,
			"block":    block,
		}).Error("Block proposer and block signer do not match.")
		return ErrInvalidBlockProposer
	}
	return nil
}

func recoverSignerFromSignature(alg algorithm.Algorithm, plainText []byte, cipherText []byte) (common.Address, error) {
	signature, err := crypto.NewSignature(alg)
	if err != nil {
		return common.Address{}, err
	}
	pub, err := signature.RecoverPublic(plainText, cipherText)
	if err != nil {
		return common.Address{}, err
	}
	addr, err := common.PublicKeyToAddress(pub)
	if err != nil {
		return common.Address{}, err
	}
	return addr, nil
}

func findProposer(ts int64, miners []*common.Address) (proposer *common.Address, err error) {
	now := time.Duration(ts) * time.Second
	if now%BlockInterval != 0 {
		return nil, ErrInvalidBlockForgeTime
	}
	offsetInDynastyInterval := now % DynasyInterval
	offsetInDynasty := offsetInDynastyInterval % DynastySize

	if int(offsetInDynasty) >= len(miners) {
		logging.WithFields(logrus.Fields{
			"offset": offsetInDynasty,
			"miners": len(miners),
		}).Error("No proposer selected for this turn.")
		return nil, ErrFoundNilProposer
	}
	return miners[offsetInDynasty], nil
}

func (d *Dpos) mintBlock(now time.Time) error {
	tail := d.bm.TailBlock()

	deadline, err := checkDeadline(tail, now)
	if err != nil {
		return err
	}

	members, err := tail.State().Dynasty()
	if err != nil {
		return err
	}

	proposer, err := findProposer(deadline.Unix(), members)
	if err != nil {
		return err
	}

	if !d.miner.Equals(*proposer) {
		return ErrInvalidBlockProposer
	}

	block, err := d.makeBlock(tail, deadline)
	if err != nil {
		return err
	}

	// TODO @cl9200 Return transactions if an error condition.

	// TODO @cl9200 Skip verification of mint block.
	err = d.bm.PushBlockData(block.GetBlockData())
	if err != nil {
		return err
	}

	d.bm.BroadCast(block.GetBlockData())
	return nil
}

func (d *Dpos) makeBlock(tail *core.Block, deadline time.Time) (*core.Block, error) {
	block, err := core.NewBlock(d.bm.ChainID(), d.miner, tail)
	if err != nil {
		return nil, err
	}

	for deadline.Sub(time.Now()) > 0 {
		tx := d.tm.Pop()
		if tx == nil {
			break
		}
		err = block.ExecuteTransaction(tx)
		if err != nil {
			return nil, err
		}
		err = block.AcceptTransaction(tx)
		if err != nil {
			return nil, err
		}
	}
	return block, nil
}

func lastMintSlot(ts time.Time) time.Time {
	now := time.Duration(ts.Unix()) * time.Second
	last := int64(((now - time.Second) / BlockInterval) * BlockInterval)
	return time.Unix(last, 0)
}

func nextMintSlot(ts time.Time) time.Time {
	now := time.Duration(ts.Unix()) * time.Second
	next := int64(((now + BlockInterval - time.Second) / BlockInterval) * BlockInterval)
	return time.Unix(next, 0)
}

func mintDeadline(ts time.Time) time.Time {
	// TODO @cl9200 Do we need MaxMintDuration?
	return nextMintSlot(ts)
}

func checkDeadline(tail *core.Block, ts time.Time) (deadline time.Time, err error) {
	last := lastMintSlot(ts)
	next := nextMintSlot(ts)
	if tail.Timestamp() >= next.Unix() {
		return time.Time{}, ErrBlockMintedInNextSlot
	}
	if tail.Timestamp() == last.Unix() {
		return mintDeadline(ts), nil
	}
	if next.Sub(ts) < MinMintDuration {
		return mintDeadline(ts), nil
	}
	return time.Time{}, ErrWaitingBlockInLastSlot
}

func (d *Dpos) loop() {
	logging.Console().Info("Started Dpos Mining.")
	ticker := time.NewTicker(miningTickInterval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			d.mintBlock(now)
		case <-d.quitCh:
			logging.Console().Info("Stopped Dpos Mining.")
			return
		}
	}
}
