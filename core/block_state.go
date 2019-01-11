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
	txState   *coreState.TransactionState
	dposState *dState.State

	storage storage.Storage
}

//Timestamp returns timestamp
func (bs *BlockState) Timestamp() int64 {
	return bs.timestamp
}

//SetTimestamp sets timestamp
func (bs *BlockState) SetTimestamp(timestamp int64) {
	bs.timestamp = timestamp
}

// Supply returns supply in state
func (bs *BlockState) Supply() *util.Uint128 {
	return bs.supply
}

// Reward returns reward in state
func (bs *BlockState) Reward() *util.Uint128 {
	return bs.reward
}

//CPUPrice returns cpuPrice
func (bs *BlockState) CPUPrice() *util.Uint128 {
	return bs.cpuPrice
}

//CPUUsage returns cpuUsage
func (bs *BlockState) CPUUsage() uint64 {
	return bs.cpuUsage
}

//NetPrice returns netPrice
func (bs *BlockState) NetPrice() *util.Uint128 {
	return bs.netPrice
}

//NetUsage returns netUsage
func (bs *BlockState) NetUsage() uint64 {
	return bs.netUsage
}

// DposState returns dpos state in state
func (bs *BlockState) DposState() *dState.State {
	return bs.dposState
}

// GetDynasty returns list of dynasty (only used in grpc)
func (bs *BlockState) GetDynasty() ([]common.Address, error) { // TODO: deprecate ?

	return bs.DposState().Dynasty()
}

//Price returns cpu price and net price
func (bs *BlockState) Price() common.Price {
	return common.Price{CPUPrice: bs.cpuPrice, NetPrice: bs.netPrice}
}

func newStates(consensus Consensus, stor storage.Storage) (*BlockState, error) {
	accState, err := coreState.NewAccountState(nil, stor)
	if err != nil {
		return nil, err
	}

	txState, err := coreState.NewTransactionState(nil, stor)
	if err != nil {
		return nil, err
	}

	dposState, err := consensus.NewConsensusState(nil, stor)
	if err != nil {
		return nil, err
	}

	return &BlockState{
		reward:    util.NewUint128(),
		supply:    util.NewUint128(),
		cpuPrice:  util.NewUint128(),
		cpuUsage:  0,
		netPrice:  util.NewUint128(),
		netUsage:  0,
		accState:  accState,
		txState:   txState,
		dposState: dposState,
		storage:   stor,
	}, nil
}

//Clone clone states
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
		storage:   bs.storage,
	}, nil
}

func (bs *BlockState) prepare() error {
	if err := bs.accState.Prepare(); err != nil {
		return err
	}
	if err := bs.txState.Prepare(); err != nil {
		return err
	}
	if err := bs.DposState().Prepare(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) beginBatch() error {
	if err := bs.accState.BeginBatch(); err != nil {
		return err
	}
	if err := bs.txState.BeginBatch(); err != nil {
		return err
	}
	if err := bs.DposState().BeginBatch(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) commit() error {
	if err := bs.accState.Commit(); err != nil {
		return err
	}
	if err := bs.txState.Commit(); err != nil {
		return err
	}
	if err := bs.dposState.Commit(); err != nil {
		return err
	}
	return nil
}
func (bs *BlockState) rollBack() error {
	if err := bs.accState.RollBack(); err != nil {
		return err
	}
	if err := bs.txState.RollBack(); err != nil {
		return err
	}
	if err := bs.dposState.RollBack(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) flush() error {
	if err := bs.accState.Flush(); err != nil {
		return err
	}
	if err := bs.txState.Flush(); err != nil {
		return err
	}
	if err := bs.dposState.Flush(); err != nil {
		return err
	}
	return nil
}

func (bs *BlockState) reset() error {
	if err := bs.accState.Reset(); err != nil {
		return err
	}
	if err := bs.txState.Reset(); err != nil {
		return err
	}
	if err := bs.dposState.Reset(); err != nil {
		return err
	}
	return nil
}

//AccountsRoot returns account state root
func (bs *BlockState) AccountsRoot() ([]byte, error) {
	return bs.accState.RootHash()
}

//TxsRoot returns transaction state root
func (bs *BlockState) TxsRoot() ([]byte, error) {
	return bs.txState.RootHash()
}

//DposRoot returns dpos state root
func (bs *BlockState) DposRoot() ([]byte, error) {
	return bs.dposState.RootBytes()
}

//String returns stringified blocks state
func (bs *BlockState) String() string {
	return fmt.Sprintf(
		"{reward: %v, supply: %v, cpuPrice: %v, cpuPoints: %v, netPrice: %v, netPoints: %v}",
		bs.reward, bs.supply, bs.cpuPrice, bs.cpuUsage, bs.netPrice, bs.netUsage)
}

func (bs *BlockState) loadAccountState(rootHash []byte) error {
	accState, err := coreState.NewAccountState(rootHash, bs.storage)
	if err != nil {
		return err
	}
	bs.accState = accState
	return nil
}

func (bs *BlockState) loadTransactionState(rootBytes []byte) error {
	txState, err := coreState.NewTransactionState(rootBytes, bs.storage)
	if err != nil {
		return err
	}
	bs.txState = txState
	return nil
}

//GetAccount returns account in state
func (bs *BlockState) GetAccount(addr common.Address) (*coreState.Account, error) {
	return bs.accState.GetAccount(addr, bs.Timestamp())
}

//PutAccount put account to state
func (bs *BlockState) PutAccount(acc *coreState.Account) error {
	return bs.accState.PutAccount(acc)
}

//GetAccountByAlias returns account by alias name
func (bs *BlockState) GetAccountByAlias(alias string) (*coreState.Account, error) {
	return bs.accState.GetAccountByAlias(alias, bs.timestamp)
}

//PutAccountAlias set alias name for account
func (bs *BlockState) PutAccountAlias(alias string, addr common.Address) error {
	return bs.accState.PutAccountAlias(alias, addr)
}

//DelAccountAlias delete alias information from account state
func (bs *BlockState) DelAccountAlias(alias string, addr common.Address) error {
	return bs.accState.DelAccountAlias(alias, addr)
}

//GetTx get tx from transaction state
func (bs *BlockState) GetTx(txHash []byte) (*coreState.Transaction, error) {
	return bs.txState.GetTx(txHash)
}

//PutTx put tx to state
func (bs *BlockState) PutTx(tx *coreState.Transaction) error {
	return bs.txState.Put(tx)
}

//AddVotePowerToCandidate add vote power to candidate
func (bs *BlockState) AddVotePowerToCandidate(candidateID []byte, amount *util.Uint128) error {
	return bs.dposState.AddVotePowerToCandidate(candidateID, amount)
}

//SubVotePowerToCandidate subtract vote power from candidate
func (bs *BlockState) SubVotePowerToCandidate(candidateID []byte, amount *util.Uint128) error {
	return bs.dposState.SubVotePowerToCandidate(candidateID, amount)
}

// checkNonce compare given transaction's nonce with expected account's nonce
func (bs *BlockState) checkNonce(tx *coreState.Transaction) error {
	fromAcc, err := bs.GetAccount(tx.From())
	if err != nil {
		return err
	}
	expectedNonce := fromAcc.Nonce + 1
	if tx.Nonce() > expectedNonce {
		logging.WithFields(logrus.Fields{
			"hash":        tx.Hash(),
			"nonce":       tx.Nonce(),
			"expected":    expectedNonce,
			"transaction": tx,
		}).Debug("Transaction nonce gap exist")
		return ErrLargeTransactionNonce
	} else if tx.Nonce() < expectedNonce {
		return ErrSmallTransactionNonce
	}
	return nil
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

//PayReward add reward to coinbase and update reward and supply
func (bs *BlockState) PayReward(coinbase common.Address, parentSupply *util.Uint128) error {
	reward, err := calcMintReward(parentSupply)
	if err != nil {
		return err
	}

	acc, err := bs.GetAccount(coinbase)
	if err != nil {
		return err
	}
	acc.Balance, err = acc.Balance.Add(reward)
	if err != nil {
		return err
	}
	err = bs.PutAccount(acc)
	if err != nil {
		return err
	}

	supply, err := parentSupply.Add(reward)
	if err != nil {
		return err
	}

	bs.reward = reward
	bs.supply = supply

	return nil
}

// AcceptTransaction consume bandwidth and adds tx in block state
func (bs *BlockState) AcceptTransaction(tx *coreState.Transaction) error {
	if tx.Receipt() == nil {
		return ErrNoTransactionReceipt
	}

	// consume payer's points
	payer, err := bs.GetAccount(tx.Payer())
	if err != nil {
		return err
	}

	payer.Points, err = payer.Points.Sub(tx.Receipt().Points())
	if err == util.ErrUint128Underflow {
		logging.Console().WithFields(logrus.Fields{
			"tx_points":    tx.Receipt().Points(),
			"payer_points": payer.Points,
			"payer":        tx.Payer().Hex(),
			"err":          err,
		}).Warn("Points limit exceeded.")
		return coreState.ErrPointNotEnough
	}
	if err != nil {
		return err
	}

	if err := bs.PutAccount(payer); err != nil {
		return err
	}

	// increase from's points
	from, err := bs.GetAccount(tx.From())
	if err != nil {
		return err
	}
	from.Nonce++
	if err := bs.PutAccount(from); err != nil {
		return err
	}

	bs.cpuUsage += tx.Receipt().CPUUsage()
	bs.netUsage += tx.Receipt().NetUsage()

	if err := bs.PutTx(tx); err != nil {
		return err
	}

	return nil
}

//SetMintDynastyState set mint dys
func (bs *BlockState) SetMintDynastyState(parentState *BlockState, consensus Consensus) error {
	mintDynasty, err := consensus.MakeMintDynasty(bs.timestamp, parentState)
	if err != nil {
		return err
	}
	return bs.dposState.SetDynasty(mintDynasty)
}
