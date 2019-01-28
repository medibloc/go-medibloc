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
	"fmt"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	dState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// BlockState is block state
type BlockState struct {
	timestamp int64

	reward   *util.Uint128
	supply   *util.Uint128
	cpuPrice *util.Uint128
	cpuUsage uint64
	netPrice *util.Uint128
	netUsage uint64

	accState  *coreState.AccountState
	txState   *TransactionState
	dposState *dState.State
}

func NewBlockState(bd *BlockData, consensus Consensus, stor storage.Storage) (*BlockState, error) {
	accState, err := coreState.NewAccountState(bd.AccStateRoot(), stor)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": bd,
			"err":   err,
		}).Error("Failed to create account state.")
		return nil, err
	}
	txState, err := NewTransactionState(bd.TxStateRoot(), stor)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": bd,
			"err":   err,
		}).Error("Failed to create transaction state.")
		return nil, err
	}
	dposState, err := consensus.NewConsensusState(bd.DposRoot(), stor)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"block": bd,
			"err":   err,
		}).Error("Failed to create consensus state.")
		return nil, err
	}
	return &BlockState{
		timestamp: bd.timestamp,
		reward:    bd.Reward().DeepCopy(),
		supply:    bd.Supply().DeepCopy(),
		cpuPrice:  bd.CPUPrice().DeepCopy(),
		cpuUsage:  bd.CPUUsage(),
		netPrice:  bd.NetPrice().DeepCopy(),
		netUsage:  bd.NetUsage(),
		accState:  accState,
		txState:   txState,
		dposState: dposState,
	}, nil
}

// Clone clone states
func (bs *BlockState) Clone() (*BlockState, error) {
	accState, err := bs.accState.Clone()
	if err != nil {
		return nil, err
	}

	txState, err := bs.txState.Clone()
	if err != nil {
		return nil, err
	}

	dposState, err := bs.dposState.Clone()
	if err != nil {
		return nil, err
	}

	return &BlockState{
		timestamp: bs.timestamp,
		reward:    bs.reward.DeepCopy(),
		supply:    bs.supply.DeepCopy(),
		cpuPrice:  bs.cpuPrice.DeepCopy(),
		cpuUsage:  bs.cpuUsage,
		netPrice:  bs.netPrice.DeepCopy(),
		netUsage:  bs.netUsage,
		accState:  accState,
		txState:   txState,
		dposState: dposState,
	}, nil
}

func (bs *BlockState) applyBatchToEachState(fn trie.BatchCallType) error {
	err := fn(bs.accState)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to apply batch function to account state.")
		return err
	}
	err = fn(bs.txState)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to apply batch function to transaction state.")
		return err
	}
	err = fn(bs.dposState)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to apply batch function to dpos state.")
		return err
	}
	return nil
}

func (bs *BlockState) prepare() error {
	err := bs.applyBatchToEachState(trie.PrepareCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to prepare block state.")
		return err
	}
	return nil
}

func (bs *BlockState) beginBatch() error {
	err := bs.applyBatchToEachState(trie.BeginBatchCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to begin batch block state.")
		return err
	}
	return nil
}

func (bs *BlockState) commit() error {
	err := bs.applyBatchToEachState(trie.CommitCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to commit block state.")
		return err
	}
	return nil
}
func (bs *BlockState) rollBack() error {
	err := bs.applyBatchToEachState(trie.RollbackCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to rollback block state.")
		return err
	}
	return nil
}

func (bs *BlockState) flush() error {
	err := bs.applyBatchToEachState(trie.FlushCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to flush block state.")
		return err
	}
	return nil
}

func (bs *BlockState) reset() error {
	err := bs.applyBatchToEachState(trie.ResetCall)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to reset block state.")
		return err
	}
	return nil
}

// String returns stringified blocks state
func (bs *BlockState) String() string {
	return fmt.Sprintf(
		"{reward: %v, supply: %v, cpuPrice: %v, cpuPoints: %v, netPrice: %v, netPoints: %v}",
		bs.reward, bs.supply, bs.cpuPrice, bs.cpuUsage, bs.netPrice, bs.netUsage)
}

// Timestamp returns timestamp
func (bs *BlockState) Timestamp() int64 {
	return bs.timestamp
}

// SetTimestamp sets timestamp
func (bs *BlockState) SetTimestamp(timestamp int64) {
	bs.timestamp = timestamp
}

// Supply returns supply in state
func (bs *BlockState) Supply() *util.Uint128 {
	return bs.supply
}

func (bs *BlockState) SetSupply(supply *util.Uint128) {
	bs.supply = supply
}

// Reward returns reward in state
func (bs *BlockState) Reward() *util.Uint128 {
	return bs.reward
}

func (bs *BlockState) SetReward(reward *util.Uint128) {
	bs.reward = reward
}

// CPUPrice returns cpuPrice
func (bs *BlockState) CPUPrice() *util.Uint128 {
	return bs.cpuPrice
}

// CPUUsage returns cpuUsage
func (bs *BlockState) CPUUsage() uint64 {
	return bs.cpuUsage
}

// NetPrice returns netPrice
func (bs *BlockState) NetPrice() *util.Uint128 {
	return bs.netPrice
}

// NetUsage returns netUsage
func (bs *BlockState) NetUsage() uint64 {
	return bs.netUsage
}

// DposState returns dpos state in state
func (bs *BlockState) DposState() *dState.State {
	return bs.dposState
}

// Price returns cpu price and net price
func (bs *BlockState) Price() common.Price {
	return common.Price{CPUPrice: bs.cpuPrice, NetPrice: bs.netPrice}
}

// AccountsRoot returns account state root
func (bs *BlockState) AccountsRoot() ([]byte, error) {
	return bs.accState.RootHash()
}

// TxsRoot returns transaction state root
func (bs *BlockState) TxsRoot() ([]byte, error) {
	return bs.txState.RootHash()
}

// DposRoot returns dpos state root
func (bs *BlockState) DposRoot() ([]byte, error) {
	return bs.dposState.RootBytes()
}

// GetAccount returns account in state
func (bs *BlockState) GetAccount(addr common.Address) (*coreState.Account, error) {
	return bs.accState.GetAccount(addr, bs.Timestamp())
}

// PutAccount put account to state
func (bs *BlockState) PutAccount(acc *coreState.Account) error {
	return bs.accState.PutAccount(acc)
}

// GetAccountByAlias returns account by alias name
func (bs *BlockState) GetAccountByAlias(alias string) (*coreState.Account, error) {
	return bs.accState.GetAccountByAlias(alias, bs.timestamp)
}

// PutAccountAlias set alias name for account
func (bs *BlockState) PutAccountAlias(alias string, addr common.Address) error {
	return bs.accState.PutAccountAlias(alias, addr)
}

// DelAccountAlias delete alias information from account state
func (bs *BlockState) DelAccountAlias(alias string, addr common.Address) error {
	return bs.accState.DelAccountAlias(alias, addr)
}

// GetTx get tx from transaction state
func (bs *BlockState) GetTx(txHash []byte) (*Transaction, error) {
	return bs.txState.GetTx(txHash)
}

// PutTx put tx to state
func (bs *BlockState) PutTx(tx *Transaction) error {
	return bs.txState.Put(tx)
}

// AddVotePowerToCandidate add vote power to candidate
func (bs *BlockState) AddVotePowerToCandidate(candidateID []byte, amount *util.Uint128) error {
	return bs.dposState.AddVotePowerToCandidate(candidateID, amount)
}

// SubVotePowerToCandidate subtract vote power from candidate
func (bs *BlockState) SubVotePowerToCandidate(candidateID []byte, amount *util.Uint128) error {
	return bs.dposState.SubVotePowerToCandidate(candidateID, amount)
}

func (bs *BlockState) checkBandwidthLimit(bandwidth *common.Bandwidth) error {
	cpu := bandwidth.CPUUsage()
	net := bandwidth.NetUsage()

	blockCPUUsage := bs.cpuUsage + cpu
	if CPULimit < blockCPUUsage {
		logging.Console().WithFields(logrus.Fields{
			"currentCPUUsage": blockCPUUsage,
			"maxCPUUsage":     CPULimit,
			"tx_cpu":          cpu,
		}).Info("Not enough block cpu bandwidth to accept transaction")
		return ErrExceedBlockMaxCPUUsage
	}

	blockNetUsage := bs.netUsage + net
	if NetLimit < blockNetUsage {
		logging.Console().WithFields(logrus.Fields{
			"currentNetUsage": blockNetUsage,
			"maxNetUsage":     NetLimit,
			"tx_net":          net,
		}).Info("Not enough block net bandwidth to accept transaction")
		return ErrExceedBlockMaxNetUsage
	}
	return nil
}
