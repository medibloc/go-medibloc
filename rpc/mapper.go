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

package rpc

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

func coreAccount2rpcAccount(account *core.Account, address string) *rpcpb.GetAccountResponse {
	return &rpcpb.GetAccountResponse{
		Address:       address,
		Balance:       account.Balance.String(),
		Nonce:         account.Nonce,
		Vesting:       account.Vesting.String(),
		Voted:         byteutils.BytesSlice2HexSlice(account.VotedSlice()),
		Records:       byteutils.BytesSlice2HexSlice(account.RecordsSlice()),
		CertsIssued:   byteutils.BytesSlice2HexSlice(account.CertsIssuedSlice()),
		CertsReceived: byteutils.BytesSlice2HexSlice(account.CertsReceivedSlice()),
		TxsFrom:       byteutils.BytesSlice2HexSlice(account.TxsFromSlice()),
		TxsTo:         byteutils.BytesSlice2HexSlice(account.TxsToSlice()),
	}
}

func coreBlock2rpcBlock(block *core.Block) (*rpcpb.GetBlockResponse, error) {
	tx, err := coreTxs2rpcTxs(block.Transactions(), true)
	if err != nil {
		return nil, err
	}

	return &rpcpb.GetBlockResponse{
		Height:       block.Height(),
		Hash:         byteutils.Bytes2Hex(block.Hash()),
		ParentHash:   byteutils.Bytes2Hex(block.ParentHash()),
		Coinbase:     block.Coinbase().Hex(),
		Reward:       block.Reward().String(),
		Supply:       block.Supply().String(),
		Timestamp:    block.Timestamp(),
		ChainId:      block.ChainID(),
		Alg:          uint32(block.Alg()),
		Sign:         byteutils.Bytes2Hex(block.Sign()),
		AccsRoot:     byteutils.Bytes2Hex(block.AccStateRoot()),
		DataRoot:     byteutils.Bytes2Hex(block.DataStateRoot()),
		DposRoot:     byteutils.Bytes2Hex(block.DposRoot()),
		UsageRoot:    byteutils.Bytes2Hex(block.UsageRoot()),
		Transactions: tx,
	}, nil
}

func coreCandidate2rpcCandidate(candidate *core.Account) *rpcpb.Candidate {
	return &rpcpb.Candidate{
		Address:   candidate.Address.Hex(),
		Collatral: candidate.Collateral.String(),
		VotePower: candidate.VotePower.String(),
	}
}

func coreTx2rpcTx(tx *core.Transaction, executed bool) (*rpcpb.GetTransactionResponse, error) {
	return &rpcpb.GetTransactionResponse{
		Hash:      byteutils.Bytes2Hex(tx.Hash()),
		From:      tx.From().Hex(),
		To:        tx.To().Hex(),
		Value:     tx.Value().String(),
		Timestamp: tx.Timestamp(),
		TxType:    tx.TxType(),
		Nonce:     tx.Nonce(),
		ChainId:   tx.ChainID(),
		Payload:   byteutils.Bytes2Hex(tx.Payload()),
		Alg:       uint32(tx.Alg()),
		Sign:      byteutils.Bytes2Hex(tx.Sign()),
		PayerSign: byteutils.Bytes2Hex(tx.PayerSign()),
		Executed:  executed,
	}, nil
}

func coreTxs2rpcTxs(txs []*core.Transaction, executed bool) ([]*rpcpb.GetTransactionResponse, error) {
	var rpcTxs []*rpcpb.GetTransactionResponse
	for _, tx := range txs {
		rpcTx, err := coreTx2rpcTx(tx, executed)
		if err != nil {
			return nil, err
		}
		rpcTxs = append(rpcTxs, rpcTx)
	}
	return rpcTxs, nil
}
