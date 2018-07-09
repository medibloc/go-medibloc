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

package dpos

import (
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
)

// Dpos returns dpos consensus model.
type Dpos struct {
	dynastySize int

	coinbase common.Address
	miner    common.Address
	minerKey signature.PrivateKey

	bm *core.BlockManager
	tm *core.TransactionManager

	genesis       *corepb.Genesis
	consensusSize int

	quitCh chan int
}

func (d *Dpos) DynastySize() int {
	return d.dynastySize
}

// New returns dpos consensus.
func New() *Dpos {
	return &Dpos{
		quitCh: make(chan int, 1),
	}
}

// NewConsensusState generates new dpos state
func (d *Dpos) NewConsensusState(dposRootBytes []byte, stor storage.Storage) (core.DposState, error) {
	pbState := new(dpospb.State)
	err := proto.Unmarshal(dposRootBytes, pbState)
	if err != nil {
		return nil, err
	}
	return NewDposState(pbState.CandidateRootHash, pbState.DynastyRootHash, stor)
}

// LoadConsensusState loads a consensus state from marshalled bytes
func (d *Dpos) LoadConsensusState(dposRootBytes []byte, stor storage.Storage) (core.DposState, error) {
	pbState := new(dpospb.State)
	err := proto.Unmarshal(dposRootBytes, pbState)
	if err != nil {
		return nil, err
	}
	return NewDposState(pbState.CandidateRootHash, pbState.DynastyRootHash, stor)
}

// Setup sets up dpos.
func (d *Dpos) Setup(cfg *medletpb.Config, genesis *corepb.Genesis, bm *core.BlockManager, tm *core.TransactionManager) error {
	// Setup miner
	if cfg.Chain.StartMine {
		d.coinbase = common.HexToAddress(cfg.Chain.Coinbase)
		d.miner = common.HexToAddress(cfg.Chain.Miner)
		minerKey, err := secp256k1.NewPrivateKeyFromHex(cfg.Chain.Privkey)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Invalid miner private key.")
			return err
		}
		d.minerKey = minerKey
	}

	// Setup genesis configuration
	d.genesis = genesis
	d.dynastySize = int(d.genesis.Meta.DynastySize)
	if d.dynastySize < 3 || d.dynastySize > 21 || d.dynastySize%3 != 0 {
		logging.Console().WithFields(logrus.Fields{
			"dynastySize": d.dynastySize,
		}).Error("Dynasty size should be a multiple of three.")
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
	newTail = bc.MainTailBlock()
	tails := bc.TailBlocks()
	for _, block := range tails {
		if bc.IsForkedBeforeLIB(block) {
			logging.WithFields(logrus.Fields{
				"block": block,
				"lib":   bc.LIB(),
			}).Debug("Blocks forked before LIB can not be selected.")
			continue
		}
		if block.Height() > newTail.Height() {
			newTail = block
		}
	}

	return newTail
}

// FindLIB finds new LIB.
func (d *Dpos) FindLIB(bc *core.BlockChain) (newLIB *core.Block) {
	lib := bc.LIB()
	tail := bc.MainTailBlock()
	cur := tail
	confirmed := make(map[string]bool)
	ds := tail.State().DposState().DynastyState()
	members, err := dynastyStateToDynasty(ds)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":  err,
			"tail": tail,
			"lib":  lib,
		}).Error("Failed to get members of dynasty.")
		return lib
	}
	dynastyGen := int64(-1)

	for !byteutils.Equal(cur.Hash(), lib.Hash()) {
		if gen := dynastyGenByTime(cur.Timestamp()); dynastyGen != gen {
			dynastyGen = gen
			confirmed = make(map[string]bool)
			ds := cur.State().DposState().DynastyState()
			members, err = dynastyStateToDynasty(ds)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"block": cur,
					"err":   err,
				}).Error("Failed to get members of dynasty.")
				return lib
			}
		}

		if cur.Height()-lib.Height() < uint64(d.consensusSize-len(confirmed)) {
			return lib
		}

		proposer, err := findProposer(cur.State().DposState(), cur.Timestamp())
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"block":     cur,
				"timestamp": cur.Timestamp(),
				"members":   members,
				"err":       err,
			}).Error("Failed to find a block proposer.")
			return lib
		}

		confirmed[proposer.Hex()] = true
		if len(confirmed) >= d.consensusSize {
			return cur
		}

		cur = bc.BlockByHash(cur.ParentHash())
		if cur == nil {
			logging.Console().WithFields(logrus.Fields{
				"tail": tail,
				"lib":  lib,
			}).Error("Failed to find a parent block.")
			return lib
		}
	}
	return lib
}

func dynastyGenByTime(ts int64) int64 {
	now := time.Duration(ts) * time.Second
	return int64(now / DynastyInterval)
}

// VerifyInterval verifies block interval.
func (d *Dpos) VerifyInterval(bd *core.BlockData, parent *core.Block) error {
	elapsed := time.Duration(bd.Timestamp()-parent.Timestamp()) * time.Second
	if elapsed%BlockInterval != 0 {
		logging.WithFields(logrus.Fields{
			"blocData":  bd,
			"timestamp": bd.Timestamp(),
		}).Debug("Invalid block interval.")
		return ErrInvalidBlockInterval
	}
	return nil
}

// VerifyProposer verifies block proposer.
func (d *Dpos) VerifyProposer(bd *core.BlockData, parent *core.Block) error {
	proposer,err  := d.FindMintProposer(bd.Timestamp(), parent)
	if err != nil {
		return err
	}
	err = verifyBlockSign(proposer, bd)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"proposer":  proposer,
			"blockData": bd,
		}).Debug("Failed to verify a block sign.")
		return err
	}
	return nil
}

func verifyBlockSign(proposer common.Address, bd *core.BlockData) error {
	signer, err := recoverSignerFromSignature(bd.Alg(), bd.Hash(), bd.Signature())
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":      err,
			"proposer": proposer,
			"signer":   signer,
			"block":    bd,
		}).Debug("Failed to recover block's signer.")
		return err
	}
	if !proposer.Equals(signer) {
		logging.WithFields(logrus.Fields{
			"signer":   signer,
			"proposer": proposer,
			"block":    bd,
		}).Debug("Block proposer and block signer do not match.")
		return ErrInvalidBlockProposer
	}
	return nil
}

func recoverSignerFromSignature(alg algorithm.Algorithm, plainText []byte, cipherText []byte) (common.Address, error) {
	sig, err := crypto.NewSignature(alg)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":       err,
			"algorithm": alg,
		}).Debug("Invalid sign algorithm.")
		return common.Address{}, err
	}
	pub, err := sig.RecoverPublic(plainText, cipherText)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err":    err,
			"plain":  byteutils.Bytes2Hex(plainText),
			"cipher": byteutils.Bytes2Hex(cipherText),
		}).Debug("Failed to recover public key from cipher text.")
		return common.Address{}, err
	}
	addr, err := common.PublicKeyToAddress(pub)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"err": err,
			"pub": pub,
		}).Debug("Failed to convert public key to address.")
		return common.Address{}, err
	}
	return addr, nil
}

func (d *Dpos) mintBlock(now time.Time) error {
	tail := d.bm.TailBlock()

	deadline, err := CheckDeadline(tail, now)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"lastSlot": lastMintSlot(now),
			"nextSlot": nextMintSlot2(now.Unix()),
			"now":      now,
			"err":      err,
		}).Debug("It's not time to mint.")
		return err
	}

	mintProposer, err := d.FindMintProposer(deadline.Unix(), tail)
	if err != nil {
		return err
	}
	if !d.miner.Equals(mintProposer) {
		logging.WithFields(logrus.Fields{
			"miner":    d.miner,
			"proposer": mintProposer,
		}).Debug("It's not my turn to mint the block.")
		return ErrInvalidBlockProposer
	}

	block, err := d.makeBlock(tail, deadline)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"tail":     tail,
			"deadline": deadline,
			"err":      err,
		}).Error("Failed to make a new block.")
		return err
	}

	err = block.Seal()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": block,
			"err":   err,
		}).Error("Failed to seal a new block.")
		return err
	}

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create new crypto signature.")
		return err
	}
	sig.InitSign(d.minerKey)

	err = block.SignThis(sig)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to sign block.")
		return err
	}

	// TODO @cl9200 Return transactions if an error condition.

	time.Sleep(deadline.Sub(time.Now()))

	logging.Console().WithFields(logrus.Fields{
		"proposer": mintProposer,
		"block":    block,
	}).Info("New block is minted.")

	// TODO @cl9200 Skip verification of mint block.
	err = d.bm.PushBlockData(block.GetBlockData())
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
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

	dynasty, err := d.makeMintBlockDynasty(deadline.Unix(), tail)
	if err != nil {
		return nil, err
	}

	if err := setDynastyState(block.State().DposState().DynastyState(), dynasty); err != nil {
		return nil, err
	}

	for deadline.Sub(time.Now()) > 0 {
		err = block.BeginBatch()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":   err,
				"block": block,
			}).Error("Failed to begin batch of new block.")
			return nil, err
		}

		transaction := d.tm.Pop()
		if transaction == nil {
			break
		}

		txMap := d.bm.TxMap()
		newTxFunc, ok := txMap[transaction.Type()]
		if !ok {
			return nil, core.ErrInvalidTransactionType
		}

		tx, err := newTxFunc(transaction)
		if err != nil {
			return nil, err
		}
		err = tx.Execute(block)

		if err != nil && err == core.ErrLargeTransactionNonce {
			if err = d.tm.Push(transaction); err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to push back tx.")
			}
			err = block.RollBack()
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to rollback new block.")
				return nil, err
			}
			continue
		}
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
				"tx":  transaction,
			}).Error("Failed to execute transaction.")

			err = block.RollBack()
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to rollback new block.")
				return nil, err
			}
			continue
		}

		err = block.AcceptTransaction(transaction)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":         err,
				"transaction": transaction,
			}).Error("Failed to accept transaction.")

			err = block.RollBack()
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to rollback new block.")
				return nil, err
			}
			continue
		}

		err = block.Commit()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":   err,
				"block": block,
			}).Error("Failed to commit new block.")
			return nil, err
		}
	}
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

func nextMintSlot2(ts int64) int64 {
	now := time.Duration(ts) * time.Second
	next := ((now + BlockInterval - time.Second) / BlockInterval) * BlockInterval
	return int64(next / time.Second)
}

func mintDeadline(ts time.Time) time.Time {
	// TODO @cl9200 Do we need MaxMintDuration?
	return nextMintSlot(ts)
}

// CheckDeadline gets deadline time of the next block to produce
func CheckDeadline(tail *core.Block, ts time.Time) (deadline time.Time, err error) {
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
