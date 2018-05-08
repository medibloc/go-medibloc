package dpos

import (
	"time"

	"errors"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Consensus properties.
const (
	BlockInterval   = 15 * time.Second
	DynastyInterval = 210 * BlockInterval
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
	ErrInvalidDynastySize     = errors.New("invalid dynasty size")
)

// Dpos returns dpos consensus model.
type Dpos struct {
	coinbase common.Address
	miner    common.Address
	minerKey signature.PrivateKey

	bm *core.BlockManager
	tm *core.TransactionManager

	genesis       *corepb.Genesis
	dynastySize   int
	consensusSize int

	quitCh chan int
}

// New returns dpos consensus.
func New(cfg *medletpb.Config) (*Dpos, error) {
	dpos := &Dpos{
		quitCh: make(chan int, 1),
	}

	if cfg.Chain.StartMine {
		dpos.coinbase = common.HexToAddress(cfg.Chain.Coinbase)
		dpos.miner = common.HexToAddress(cfg.Chain.Miner)
		minerKey, err := secp256k1.NewPrivateKeyFromHex(cfg.Chain.Privkey)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Invalid miner private key.")
			return nil, err
		}
		dpos.minerKey = minerKey
	}
	return dpos, nil
}

// Setup sets up dpos.
func (d *Dpos) Setup(genesis *corepb.Genesis, bm *core.BlockManager, tm *core.TransactionManager) error {
	d.genesis = genesis

	d.dynastySize = int(d.genesis.Meta.DynastySize)
	if d.dynastySize < 3 || d.dynastySize > 21 || d.dynastySize%3 != 0 {
		return ErrInvalidDynastySize
	}

	d.consensusSize = d.dynastySize*2/3 + 1
	d.bm = bm
	d.tm = tm
	return nil
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
		return lib
	}
	dynastyGen := int64(-1)

	for !cur.Hash().Equals(lib.Hash()) {
		if gen := dynastyGenByTime(cur.Timestamp()); dynastyGen != gen {
			dynastyGen = gen
			confirmed = make(map[string]bool)
			// TODO @cl9200 Replace member of dynasty.
			// members, err = tail.State().Dynasty()
		}

		if cur.Height()-lib.Height() < uint64(d.consensusSize-len(confirmed)) {
			return lib
		}

		proposer, err := FindProposer(cur.Timestamp(), members, d.dynastySize)
		if err != nil {
			return lib
		}

		confirmed[proposer.Hex()] = true
		if len(confirmed) >= d.consensusSize {
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
	return int64(now / DynastyInterval)
}

// VerifyProposer verifies block proposer.
func (d *Dpos) VerifyProposer(bc *core.BlockChain, block *core.BlockData) error {
	// TODO @cl9200 Handling when tail height is higher than block height.
	tail := bc.MainTailBlock()
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

	proposer, err := FindProposer(block.Timestamp(), members, d.dynastySize)
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

// FindProposer finds proposer of given timestamp.
func FindProposer(ts int64, miners []*common.Address, dynastySize int) (proposer *common.Address, err error) {
	now := time.Duration(ts) * time.Second
	if now%BlockInterval != 0 {
		return nil, ErrInvalidBlockForgeTime
	}
	offsetInDynastyInterval := (now / BlockInterval) % DynastyInterval
	offsetInDynasty := int(offsetInDynastyInterval) % dynastySize

	if offsetInDynasty >= len(miners) {
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
		logging.WithFields(logrus.Fields{
			"lastSlot": lastMintSlot(now),
			"nextSlot": nextMintSlot(now),
			"now":      now,
			"err":      err,
		}).Debug("It's not time to mint.")
		return err
	}

	members, err := tail.State().Dynasty()
	if err != nil {
		logging.WithFields(logrus.Fields{
			"members": members,
			"err":     err,
		}).Error("Failed to get dynasty members.")
		return err
	}

	proposer, err := FindProposer(deadline.Unix(), members, d.dynastySize)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"members":  members,
			"deadline": deadline,
			"err":      err,
		}).Debug("Failed to find block proposer.")
		return err
	}

	if !d.miner.Equals(*proposer) {
		logging.WithFields(logrus.Fields{
			"miner":    d.miner,
			"proposer": proposer,
		}).Debug("It's not my turn to mint the block.")
		return ErrInvalidBlockProposer
	}

	block, err := d.makeBlock(tail, deadline)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"tail":     tail,
			"deadline": deadline,
			"err":      err,
		}).Error("Failed to make a new block.")
		return err
	}

	err = block.Seal()
	if err != nil {
		logging.WithFields(logrus.Fields{
			"block": block,
			"err":   err,
		}).Error("Failed to seal a new block.")
		return err
	}

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create new crypto signature.")
		return err
	}
	sig.InitSign(d.minerKey)

	err = block.SignThis(sig)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to sign block.")
		return err
	}

	// TODO @cl9200 Return transactions if an error condition.

	time.Sleep(deadline.Sub(time.Now()))

	logging.Console().WithFields(logrus.Fields{
		"proposer": proposer,
		"block":    block,
	}).Info("New block is minted.")

	// TODO @cl9200 Skip verification of mint block.
	err = d.bm.PushBlockData(block.GetBlockData())
	if err != nil {
		logging.WithFields(logrus.Fields{
			"block": block,
			"err":   err,
		}).Error("Failed to push block to blockchain.")
		return err
	}

	d.bm.BroadCast(block.GetBlockData())

	return nil
}

func (d *Dpos) makeBlock(tail *core.Block, deadline time.Time) (*core.Block, error) {
	block, err := core.NewBlock(d.bm.ChainID(), d.miner, tail)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create new block.")
		return nil, err
	}
	block.SetTimestamp(deadline.Unix())

	block.BeginBatch()
	for deadline.Sub(time.Now()) > 0 {
		tx := d.tm.Pop()
		if tx == nil {
			break
		}
		err = block.ExecuteTransaction(tx)
		if err != nil && err == core.ErrSmallTransactionNonce {
			continue
		}
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
				"tx":  tx,
			}).Error("Failed to execute transaction.")
			block.RollBack()
			return nil, err
		}
		err = block.AcceptTransaction(tx)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
				"tx":  tx,
			}).Error("Failed to accept transaction.")
			block.RollBack()
			return nil, err
		}
	}
	block.Commit()
	return block, nil
}

func lastMintSlot(ts time.Time) time.Time {
	now := time.Duration(ts.Unix()) * time.Second
	last := ((now - time.Second) / BlockInterval) * BlockInterval
	return time.Unix(int64(last/time.Second), 0)
}

func nextMintSlot(ts time.Time) time.Time {
	now := time.Duration(ts.Unix()) * time.Second
	next := ((now + BlockInterval - time.Second) / BlockInterval) * BlockInterval
	return time.Unix(int64(next/time.Second), 0)
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
