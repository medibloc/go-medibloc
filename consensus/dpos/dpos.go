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
	"bytes"
	"io/ioutil"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Dpos returns dpos consensus model.
type Dpos struct {
	dynastySize int

	startPropose bool
	proposers    map[common.Address]*Proposer

	bm *core.BlockManager
	tm *core.TransactionManager

	quitCh chan int
}

//Proposer returns proposer
type Proposer struct {
	ProposerAddress common.Address
	Privkey         signature.PrivateKey
	Coinbase        common.Address
}

//Proposers returns Proposers
func (d *Dpos) Proposers() map[common.Address]*Proposer {
	return d.proposers
}

// DynastySize returns dynastySize
func (d *Dpos) DynastySize() int {
	return d.dynastySize
}

// New returns dpos consensus.
func New(dynastySize int) *Dpos {
	return &Dpos{
		dynastySize: dynastySize,
		quitCh:      make(chan int),
		proposers:   make(map[common.Address]*Proposer),
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
	// Setup proposer
	d.startPropose = cfg.Chain.StartMine
	if cfg.Chain.StartMine {
		if len(cfg.Chain.Proposers) == 0 {
			return ErrProposerConfigNotFound
		}
		for _, pbProposer := range cfg.Chain.Proposers {
			p := &Proposer{}
			p.Coinbase = common.HexToAddress(pbProposer.Coinbase)

			if pbProposer.Keydir != "" {
				ks, err := ioutil.ReadFile(pbProposer.Keydir)
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err":    err,
						"Keydir": pbProposer.Keydir,
					}).Error("failed to read key store file")
					return keystore.ErrFailedToReadKeystoreFile
				}

				key, err := keystore.DecryptKey(ks, pbProposer.Passphrase)
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err": err,
					}).Error("failed to decrypt keystore file")
					return keystore.ErrFailedToDecrypt
				}
				p.ProposerAddress = key.Address
				p.Privkey = key.PrivateKey

			} else {
				var err error
				p.ProposerAddress = common.HexToAddress(pbProposer.Proposer)
				p.Privkey, err = secp256k1.NewPrivateKeyFromHex(pbProposer.Privkey)
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err": err,
					}).Error("Invalid proposer private key.")
					return err
				}
			}

			logging.WithFields(logrus.Fields{
				"proposerAddr": p.ProposerAddress,
			}).Info("Proposer set complete.")

			d.proposers[p.ProposerAddress] = p
		}
	}

	// Setup genesis configuration
	d.dynastySize = int(genesis.Meta.DynastySize)
	if d.dynastySize < 3 || d.dynastySize > 21 || d.dynastySize%3 != 0 {
		logging.Console().WithFields(logrus.Fields{
			"dynastySize": d.dynastySize,
		}).Error("Dynasty size should be a multiple of three.")
		return ErrInvalidDynastySize
	}

	d.bm = bm
	d.tm = tm
	return nil
}

// Start starts proposer.
func (d *Dpos) Start() {
	if !d.startPropose {
		return
	}
	go d.loop()
}

// Stop stops proposer.
func (d *Dpos) Stop() {
	if !d.startPropose {
		return
	}
	d.quitCh <- 0
}

func (d *Dpos) consensusSize() int {
	return int(d.dynastySize*2/3 + 1)
}

//DynastyInterval returns dynasty interval
func (d *Dpos) DynastyInterval() time.Duration {
	return time.Duration(d.dynastySize*NumberOfRounds) * BlockInterval
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
	tailDynastyIndex := d.calcDynastyIndex(tail.Timestamp())

	cur := tail
	confirmed := make(map[int]bool)

	for d.calcDynastyIndex(cur.Timestamp()) == tailDynastyIndex && !bytes.Equal(cur.Hash(), lib.Hash()) {
		proposerIndex := d.calcProposerIndex(cur.Timestamp())
		confirmed[proposerIndex] = true

		if len(confirmed) >= d.consensusSize() {
			return cur
		}

		cur = bc.BlockByHash(cur.ParentHash())
		if cur == nil {
			logging.Console().WithFields(logrus.Fields{
				"cur": cur,
			}).Error("Failed to find a parent block.")
			return lib
		}
	}
	return lib
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

// VerifyProposer verifies block Proposer.
func (d *Dpos) VerifyProposer(b *core.Block) error {
	signer, err := b.BlockData.Proposer()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"blockData": b.BlockData,
		}).Warn("Failed to recover block's signer.")
		return err
	}

	proposerIndex := d.calcProposerIndex(b.Timestamp())

	proposer := new(common.Address)
	ds := b.State().DposState().DynastyState()
	if err := ds.GetData(byteutils.FromInt32(int32(proposerIndex)), proposer); err != nil {
		return err
	}
	if !signer.Equals(*proposer) {
		logging.Console().WithFields(logrus.Fields{
			"blockdata":     b.BlockData,
			"Proposer":      proposer,
			"signer":        signer,
			"proposerIndex": proposerIndex,
		}).Warn("Block Proposer and block signer do not match.")
		return ErrInvalidBlockProposer
	}
	return nil
}

func (d *Dpos) mintBlock(now time.Time) error {
	tail := d.bm.TailBlock()

	deadline, err := checkDeadline(tail, now)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"lastSlot": lastMintSlot(now),
			"nextSlot": NextMintSlot2(now.Unix()),
			"now":      now,
			"err":      err,
		}).Debug("It's not time to mint.")
		return err
	}

	// Check if it is the proposer's turn
	mintProposer, err := d.FindMintProposer(deadline.Unix(), tail)
	if err != nil {
		return err
	}
	p, ok := d.proposers[mintProposer]
	if !ok {
		logging.WithFields(logrus.Fields{
			"Proposer":    mintProposer,
			"myProposers": d.proposers,
			"test":"true",
		}).Debug("It's not my turn to mint the block.2")
		return ErrInvalidBlockProposer
	}

	block, err := d.makeBlock(p.Coinbase, tail, deadline, nextMintSlot(now))
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

	sig, err := crypto.NewSignature(algorithm.SECP256K1) // TODO @ggomma use cmd config
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create new crypto signature.")
		return err
	}
	sig.InitSign(p.Privkey)

	err = block.SignThis(sig)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to sign block.")
		return err
	}

	// TODO @cl9200 Return transactions if an error condition.

	time.Sleep(nextMintSlot(now).Sub(time.Now()))

	logging.Console().WithFields(logrus.Fields{
		"Proposer": mintProposer,
		"block":    block,
	}).Info("New block is minted.")

	err = d.bm.PushCreatedBlock(block)
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

func (d *Dpos) makeBlock(coinbase common.Address, tail *core.Block, deadline time.Time, nextMintTs time.Time) (*core.Block, error) {
	logging.Console().Info("Start to make mint block.")

	block, err := tail.Child()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to make child block for make block")
		return nil, err
	}
	block.SetTimestamp(nextMintTs.Unix())

	if err := block.Prepare(); err != nil {
		return nil, err
	}

	// Enable states changing (acc, tx, dpos)
	if err := block.BeginBatch(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to begin batch of new block.")
		block.Storage().DisableBatch()
		return nil, err
	}

	// Change dynasty state for current block
	if err := block.SetMintDynastyState(tail, d); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set dynasty")
		block.Storage().DisableBatch()
		return nil, err
	}

	// Save state changes (only dpos state in this line)
	// Even though block Proposer doesn't have enough time for including transactions below,
	// states changed above need to be safely saved.
	if err := block.Commit(); err != nil {
		block.Storage().DisableBatch()
		return nil, err
	}

	// Execute and include transactions until it's deadline
	for deadline.Sub(time.Now()) > 0 {
		logging.WithFields(logrus.Fields{
			"time.remain": deadline.Sub(time.Now()),
		}).Debug("Make block is in progress")

		// Get transaction from transaction pool
		transaction := d.tm.Pop()
		if transaction == nil {
			logging.Info("No more transactions in block pool.")
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if err := block.BeginBatch(); err != nil {
			block.Storage().DisableBatch()
			return nil, err
		}

		// Execute transaction and change states
		receipt, err := block.ExecuteTransaction(transaction, d.bm.TxMap())
		if err != nil {
			if err := block.RollBack(); err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to rollback new block.")
				block.Storage().DisableBatch()
				return nil, err
			}
			if err == core.ErrLargeTransactionNonce || err == core.ErrExceedBlockMaxCPUUsage || err == core.ErrExceedBlockMaxNetUsage {
				if err = d.tm.Push(transaction); err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err": err,
					}).Error("failed to push back transaction to tx poll")
				}
				continue
			}
		} else {
			if err := block.Commit(); err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to commit execution result")
				block.Storage().DisableBatch()
				return nil, err
			}
		}

		if receipt == nil {
			logging.Console().WithFields(logrus.Fields{
				"transaction": transaction.Hash(),
				"err":         err,
			}).Info("failed to execute transaction")
			continue
		}

		if err := block.BeginBatch(); err != nil {
			block.Storage().DisableBatch()
			return nil, err
		}

		transaction.SetReceipt(receipt)

		err = block.AcceptTransaction(transaction)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":         err,
				"transaction": transaction,
			}).Error("Failed to accept transaction.")

			if err := block.RollBack(); err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to rollback new block.")
				block.Storage().DisableBatch()
				return nil, err
			}
			continue
		}

		block.AppendTransaction(transaction)

		err = block.Commit()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":   err,
				"block": block,
			}).Error("Failed to commit new block.")
			block.Storage().DisableBatch()
			return nil, err
		}
	}

	block.BeginBatch()
	block.SetCoinbase(coinbase)
	if err := block.PayReward(coinbase, tail.Supply()); err != nil {
		return nil, err
	}
	err = block.Commit()
	if err != nil {
		return nil, err
	}

	err = block.Flush()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to flush state on storage")
		return nil, err
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

//NextMintSlot2 returns timestamp for next mint slot
func NextMintSlot2(ts int64) int64 {
	now := time.Duration(ts) * time.Second
	next := ((now + BlockInterval - time.Second) / BlockInterval) * BlockInterval
	return int64(next / time.Second)
}

func mintDeadline(ts time.Time) time.Time {
	maxMint := ts.Add(maxMintDuration)
	nextMint := nextMintSlot(ts)

	if maxMint.Unix() < nextMint.Unix() {
		return maxMint
	}
	return nextMint
}

// CheckDeadline gets deadline time of the next block to produce
func checkDeadline(tail *core.Block, ts time.Time) (deadline time.Time, err error) {
	last := lastMintSlot(ts)
	next := nextMintSlot(ts)
	if tail.Timestamp() >= next.Unix() {
		return time.Time{}, ErrBlockMintedInNextSlot
	}
	if tail.Timestamp() == last.Unix() {
		return mintDeadline(ts), nil
	}
	if next.Sub(ts) < minMintDuration {
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

func (d *Dpos) calcDynastyIndex(ts int64) int {
	return int(ts / int64(d.DynastyInterval().Seconds()))
}

func (d *Dpos) calcProposerIndex(ts int64) int {
	return (int(ts) / int(BlockInterval.Seconds())) % d.dynastySize
}
