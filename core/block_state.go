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
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// BlockState is block state
type BlockState struct {
	reward   *util.Uint128
	supply   *util.Uint128
	cpuRef   *util.Uint128
	cpuUsage *util.Uint128
	netRef   *util.Uint128
	netUsage *util.Uint128

	accState  *AccountState
	txState   *TransactionState
	dposState DposState

	storage storage.Storage
}

// Supply returns supply in state
func (bs *BlockState) Supply() *util.Uint128 {
	return bs.supply
}

// Reward returns reward in state
func (bs *BlockState) Reward() *util.Uint128 {
	return bs.reward
}

//CPURef returns cpuRef
func (bs *BlockState) CPURef() *util.Uint128 {
	return bs.cpuRef
}

//CPUUsage returns cpuUsage
func (bs *BlockState) CPUUsage() *util.Uint128 {
	return bs.cpuUsage
}

//NetRef returns netRef
func (bs *BlockState) NetRef() *util.Uint128 {
	return bs.netRef
}

//NetUsage returns netUsage
func (bs *BlockState) NetUsage() *util.Uint128 {
	return bs.netUsage
}

// AccState returns account state in state
func (bs *BlockState) AccState() *AccountState {
	return bs.accState
}

// DposState returns dpos state in state
func (bs *BlockState) DposState() DposState {
	return bs.dposState
}

// GetDynasty returns list of dynasty (only used in grpc)
func (bs *BlockState) GetDynasty() ([]common.Address, error) { // TODO: deprecate ?

	return bs.DposState().Dynasty()
}

func newStates(consensus Consensus, stor storage.Storage) (*BlockState, error) {
	accState, err := NewAccountState(nil, stor)
	if err != nil {
		return nil, err
	}

	txState, err := NewTransactionState(nil, stor)
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
		cpuRef:    util.NewUint128(),
		cpuUsage:  util.NewUint128(),
		netRef:    util.NewUint128(),
		netUsage:  util.NewUint128(),
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
		reward:    bs.reward.DeepCopy(),
		supply:    bs.supply.DeepCopy(),
		cpuRef:    bs.cpuRef.DeepCopy(),
		cpuUsage:  bs.cpuUsage.DeepCopy(),
		netRef:    bs.netRef.DeepCopy(),
		netUsage:  bs.netUsage.DeepCopy(),
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
		"{reward: %v, supply: %v, cpuRef: %v, cpuUsage: %v, netRef: %v, netUsage: %v}",
		bs.reward, bs.supply, bs.cpuRef, bs.cpuUsage, bs.netRef, bs.netUsage)
}

func (bs *BlockState) loadAccountState(rootHash []byte) error {
	accState, err := NewAccountState(rootHash, bs.storage)
	if err != nil {
		return err
	}
	bs.accState = accState
	return nil
}

func (bs *BlockState) loadTransactionState(rootBytes []byte) error {
	txState, err := NewTransactionState(rootBytes, bs.storage)
	if err != nil {
		return err
	}
	bs.txState = txState
	return nil
}

//GetAccount returns account in state
func (bs *BlockState) GetAccount(addr common.Address) (*Account, error) {
	return bs.accState.GetAccount(addr)
}

//PutAccount put account to state
func (bs *BlockState) PutAccount(acc *Account) error {
	return bs.accState.putAccount(acc)
}

//GetTx returns txs in state
func (bs *BlockState) GetTx(txHash []byte) (*Transaction, error) {
	return bs.txState.Get(txHash)
}

// checkNonce compare given transaction's nonce with expected account's nonce
func (bs *BlockState) checkNonce(tx *Transaction) error {
	fromAcc, err := bs.GetAccount(tx.From())
	if err != nil {
		return err
	}
	expectedNonce := fromAcc.Nonce + 1
	if tx.nonce > expectedNonce {
		logging.Console().WithFields(logrus.Fields{
			"hash":     tx.Hash(),
			"nonce":    tx.Nonce(),
			"expected": expectedNonce,
		}).Info("Transaction nonce gap exist")
		return ErrLargeTransactionNonce
	} else if tx.nonce < expectedNonce {
		return ErrSmallTransactionNonce
	}
	return nil
}

// checkBandwidth compare given transaction's required bandwidth with the account's remaining bandwidth
func (bs *BlockState) checkPayerBandwidth(payer *Account, transaction *Transaction, cpuUsage, netUsage *util.Uint128) error {
	avail, err := payer.Vesting.Sub(payer.Bandwidth)
	if err != nil {
		return err
	}
	switch transaction.TxType() {
	case TxOpVest:
		avail, err = avail.Add(transaction.Value())
		if err != nil {
			return err
		}
	case TxOpWithdrawVesting:
		avail, err = avail.Sub(transaction.Value())
		if err == util.ErrUint128Underflow {
			return ErrVestingNotEnough
		}
		if err != nil {
			return err
		}
	}
	usage, err := cpuUsage.Add(netUsage)
	if err != nil {
		return err
	}
	if avail.Cmp(usage) < 0 {
		return ErrBandwidthNotEnough
	}
	return nil
}

func (bs *BlockState) checkBandwidthLimit(cpu, net *util.Uint128) error {
	blockCPUUsage, err := bs.cpuUsage.Add(cpu)
	if err != nil {
		return err
	}
	maxCPU, err := bs.cpuRef.Mul(util.NewUint128FromUint(uint64(cpuLimit)))
	if err != nil {
		return err
	}
	if maxCPU.Cmp(blockCPUUsage) < 0 {
		logging.Console().WithFields(logrus.Fields{
			"currentCPUUsage": blockCPUUsage,
			"maxCPUUsage":     maxCPU,
			"tx_cpu":          cpu,
		}).Info("Not enough block cpu bandwidth to accept transaction")
		return ErrExceedBlockMaxCPUUsage
	}

	blockNetUsage, err := bs.netUsage.Add(net)
	if err != nil {
		return err
	}
	maxNet, err := bs.netRef.Mul(util.NewUint128FromUint(uint64(netLimit)))
	if err != nil {
		return err
	}
	if maxNet.Cmp(blockNetUsage) < 0 {
		logging.Console().WithFields(logrus.Fields{
			"currentNetUsage": blockNetUsage,
			"maxNetUsage":     maxNet,
			"tx_net":          net,
		}).Info("Not enough block net bandwidth to accept transaction")
		return ErrExceedBlockMaxNetUsage
	}
	return nil
}
