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
	"encoding/json"
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
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Dpos returns dpos consensus model.
type Dpos struct {
	dynastySize int

	startMine bool
	coinbase  common.Address
	miner     common.Address
	minerKey  signature.PrivateKey

	bm *core.BlockManager
	tm *core.TransactionManager

	genesis *corepb.Genesis

	quitCh chan int
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
	d.startMine = cfg.Chain.StartMine
	if cfg.Chain.StartMine {
		d.coinbase = common.HexToAddress(cfg.Chain.Coinbase)
		var address, privKey string

		if cfg.Chain.Keydir != "" {
			ksString, err := ioutil.ReadFile(cfg.Chain.Keydir)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err":    err,
					"Keydir": cfg.Chain.Keydir,
				}).Error("Error in reading keystore file.")
				return err
			}
			ksf := crypto.EncryptedKeyJSONV3{}
			json.Unmarshal([]byte(ksString), ksf)
			key, err := crypto.DecryptKey([]byte(ksString), cfg.Chain.Passphrase)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Error in DecryptKey.")
				return err
			}
			pk := secp256k1.NewPrivateKey(key.PrivateKey)
			addr, err := common.PublicKeyToAddress(pk.PublicKey())
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Error in PublicKeyToAddress.")
				return err
			}
			address = byteutils.Bytes2Hex(addr.Bytes())
			privKey = byteutils.Bytes2Hex(secp256k1.FromECDSAPrivateKey(key.PrivateKey))
		} else {
			address = cfg.Chain.Miner
			privKey = cfg.Chain.Privkey
		}
		d.miner = common.HexToAddress(address)
		minerKey, err := secp256k1.NewPrivateKeyFromHex(privKey)
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

	d.bm = bm
	d.tm = tm
	return nil
}

// Start starts miner.
func (d *Dpos) Start() {
	if !d.startMine {
		return
	}
	go d.loop()
}

// Stop stops miner.
func (d *Dpos) Stop() {
	if !d.startMine {
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

// VerifyProposer verifies block proposer.
func (d *Dpos) VerifyProposer(bd *core.BlockData, parent *core.Block) error {
	signer, err := bd.Proposer()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":       err,
			"blockData": bd,
		}).Warn("Failed to recover block's signer.")
		return err
	}

	proposer, err := d.FindMintProposer(bd.Timestamp(), parent)
	if err != nil {
		return err
	}
	if !signer.Equals(proposer) {

		dynasty, _ := d.MakeMintDynasty(bd.Timestamp(), parent)

		logging.Console().WithFields(logrus.Fields{
			"bd":       bd,
			"proposer": proposer,
			"signer":   signer,
			"dynasty":  dynasty,
		}).Warn("Block proposer and block signer do not match.")
		return ErrInvalidBlockProposer
	}
	return nil
}

func (d *Dpos) mintBlock(now time.Time) error {
	tail := d.bm.TailBlock()

	deadline, err := CheckDeadline(tail, now)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"lastSlot": lastMintSlot(now),
			"nextSlot": NextMintSlot2(now.Unix()),
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

	block, err := d.makeBlock(tail, deadline, nextMintSlot(now))
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

	time.Sleep(nextMintSlot(now).Sub(time.Now()))

	logging.Console().WithFields(logrus.Fields{
		"proposer": mintProposer,
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

func (d *Dpos) makeBlock(tail *core.Block, deadline time.Time, nextMintTs time.Time) (*core.Block, error) {
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

	if err := block.BeginBatch(); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err":   err,
			"block": block,
		}).Error("Failed to begin batch of new block.")
		block.Storage().DisableBatch()
		return nil, err
	}

	if err := block.SetMintDynastyState(tail); err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set dynasty")
		block.Storage().DisableBatch()
		return nil, err
	}

	if err := block.Commit(); err != nil {
		block.Storage().DisableBatch()
		return nil, err
	}

	for deadline.Sub(time.Now()) > 0 {
		logging.WithFields(logrus.Fields{
			"time.remain": deadline.Sub(time.Now()),
		}).Debug("Make block is in progress")
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
		err = block.ExecuteTransaction(transaction, d.bm.TxMap())

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
				block.Storage().DisableBatch()
				return nil, err
			}
			continue
		}
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
				"tx":  transaction,
			}).Error("Failed to execute transaction.")
			// TODO : handle failed transaction (event?, or accept & write on receipt)
			err = block.RollBack()
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed to rollback new block.")
				block.Storage().DisableBatch()
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
				block.Storage().DisableBatch()
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
			block.Storage().DisableBatch()
			return nil, err
		}
	}

	block.BeginBatch()
	block.SetCoinbase(d.coinbase)
	if err := block.PayReward(d.coinbase, tail.Supply()); err != nil {
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
func CheckDeadline(tail *core.Block, ts time.Time) (deadline time.Time, err error) {
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
