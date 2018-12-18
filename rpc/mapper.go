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
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	rpcpb "github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func coreAccount2rpcAccount(account *core.Account, curTs int64, address string) (*rpcpb.Account, error) {
	if err := account.UpdateBandwidth(curTs); err != nil {
		return nil, status.Error(codes.Internal, ErrMsgFailedToUpdateBandwidth)
	}
	if err := account.UpdateUnstaking(curTs); err != nil {
		return nil, status.Error(codes.Internal, ErrMsgFailedToUpdateUnstaking)
	}
	aliasBytes, err := account.GetData(core.AliasPrefix, []byte(core.AliasKey))
	if err != nil && err != core.ErrNotFound {
		return nil, err
	}
	pbAlias := new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		return nil, err
	}
	return &rpcpb.Account{
		Address:     address,
		Balance:     account.Balance.String(),
		Nonce:       account.Nonce,
		Vesting:     account.Vesting.String(),
		Voted:       byteutils.BytesSlice2HexSlice(account.VotedSlice()),
		Bandwidth:   account.Bandwidth.String(),
		Unstaking:   account.Unstaking.String(),
		Alias:       pbAlias.AliasName,
		CandidateId: byteutils.Bytes2Hex(account.CandidateID),
	}, nil
}

func coreBlock2rpcBlock(block *core.Block, light bool) (*rpcpb.Block, error) {
	var txs []*rpcpb.Transaction
	var txHashes []string
	var err error
	if light {
		for _, tx := range block.Transactions() {
			txHashes = append(txHashes, byteutils.Bytes2Hex(tx.Hash()))
		}
	} else {
		txs, err = coreTxs2rpcTxs(block.Transactions(), true)
		if err != nil {
			return nil, err
		}
	}

	return &rpcpb.Block{
		Height:       block.Height(),
		Hash:         byteutils.Bytes2Hex(block.Hash()),
		ParentHash:   byteutils.Bytes2Hex(block.ParentHash()),
		Coinbase:     block.Coinbase().Hex(),
		Reward:       block.Reward().String(),
		Supply:       block.Supply().String(),
		Timestamp:    block.Timestamp(),
		ChainId:      block.ChainID(),
		HashAlg:      uint32(block.HashAlg()),
		CryptoAlg:    uint32(block.CryptoAlg()),
		Sign:         byteutils.Bytes2Hex(block.Sign()),
		AccsRoot:     byteutils.Bytes2Hex(block.AccStateRoot()),
		TxsRoot:      byteutils.Bytes2Hex(block.TxStateRoot()),
		DposRoot:     byteutils.Bytes2Hex(block.DposRoot()),
		Transactions: txs,
		TxHashes:     txHashes,
	}, nil
}

func dposCandidate2rpcCandidate(candidate *dpos.Candidate) *rpcpb.Candidate {
	return &rpcpb.Candidate{
		CandidateId: byteutils.Bytes2Hex(candidate.ID),
		Address:     candidate.Addr.Hex(),
		Url:         candidate.URL,
		Collateral:  candidate.Collateral.String(),
		VotePower:   candidate.VotePower.String(),
	}
}

// CoreTx2rpcTx converts core transaction type to rpcpb response type
func CoreTx2rpcTx(tx *core.Transaction, onChain bool) (*rpcpb.Transaction, error) {
	var rpcReceipt *rpcpb.TransactionReceipt
	var err error

	if onChain {
		rpcReceipt, err = coreReceipt2rpcReceipt(tx)
		if err != nil {
			return nil, err
		}
	}

	return &rpcpb.Transaction{
		Hash:      byteutils.Bytes2Hex(tx.Hash()),
		From:      tx.From().Hex(),
		To:        tx.To().Hex(),
		Value:     tx.Value().String(),
		Timestamp: tx.Timestamp(),
		TxType:    tx.TxType(),
		Nonce:     tx.Nonce(),
		ChainId:   tx.ChainID(),
		Payload:   byteutils.Bytes2Hex(tx.Payload()),
		HashAlg:   uint32(tx.HashAlg()),
		CryptoAlg: uint32(tx.CryptoAlg()),
		Sign:      byteutils.Bytes2Hex(tx.Sign()),
		PayerSign: byteutils.Bytes2Hex(tx.PayerSign()),
		OnChain:   onChain,
		Receipt:   rpcReceipt,
	}, nil
}

func coreTxs2rpcTxs(txs []*core.Transaction, onChain bool) ([]*rpcpb.Transaction, error) {
	var rpcTxs []*rpcpb.Transaction
	for _, tx := range txs {
		rpcTx, err := CoreTx2rpcTx(tx, onChain)
		if err != nil {
			return nil, err
		}
		rpcTxs = append(rpcTxs, rpcTx)
	}
	return rpcTxs, nil
}

func coreReceipt2rpcReceipt(tx *core.Transaction) (*rpcpb.TransactionReceipt, error) {
	err := string(tx.Receipt().Error())
	cpuUsage := tx.Receipt().CPUUsage().String()
	netUsage := tx.Receipt().NetUsage().String()

	return &rpcpb.TransactionReceipt{
		Executed: tx.Receipt().Executed(),
		CpuUsage: cpuUsage,
		NetUsage: netUsage,
		Error:    err,
	}, nil
}
