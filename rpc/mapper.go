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
	"encoding/json"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func coreAccount2rpcAccount(account *core.Account, address string) *rpcpb.GetAccountResponse {
	if account == nil {
		return &rpcpb.GetAccountResponse{
			Address:       address,
			Balance:       "0",
			Nonce:         0,
			Vesting:       "0",
			Voted:         "",
			Records:       []string{},
			CertsIssued:   []string{},
			CertsReceived: []string{},
			TxsFrom:       []string{},
			TxsTo:         []string{},
		}
	}
	return &rpcpb.GetAccountResponse{
		Address:       address,
		Balance:       account.Balance().String(),
		Nonce:         account.Nonce(),
		Vesting:       account.Vesting().String(),
		Voted:         byteutils.Bytes2Hex(account.Voted()),
		Records:       nil, // TODO @ggomma
		CertsIssued:   nil, // TODO @ggomma
		CertsReceived: nil, // TODO @ggomma
		TxsFrom:       byteutils.BytesSlice2HexSlice(account.TxsFrom()),
		TxsTo:         byteutils.BytesSlice2HexSlice(account.TxsTo()),
	}
}

func coreBlock2rpcBlock(block *core.Block) *rpcpb.GetBlockResponse {
	return &rpcpb.GetBlockResponse{
		Height:            block.Height(),
		Hash:              byteutils.Bytes2Hex(block.Hash()),
		ParentHash:        byteutils.Bytes2Hex(block.ParentHash()),
		Coinbase:          block.Coinbase().Hex(),
		Reward:            block.Reward().String(),
		Supply:            block.Supply().String(),
		Timestamp:         block.Timestamp(),
		ChainId:           block.ChainID(),
		Alg:               uint32(block.Alg()),
		Sign:              byteutils.Bytes2Hex(block.Sign()),
		AccsRoot:          byteutils.Bytes2Hex(block.AccsRoot()),
		TxsRoot:           byteutils.Bytes2Hex(block.TxsRoot()),
		UsageRoot:         byteutils.Bytes2Hex(block.UsageRoot()),
		RecordsRoot:       byteutils.Bytes2Hex(block.RecordsRoot()),
		CertificationRoot: byteutils.Bytes2Hex(block.CertificationRoot()),
		DposRoot:          byteutils.Bytes2Hex(block.DposRoot()),
		Transactions:      coreTxs2rpcTxs(block.Transactions(), true),
	}
}

func dposCandidate2rpcCandidate(candidate *dpospb.Candidate) (*rpcpb.Candidate, error) {
	collatral, err := util.NewUint128FromFixedSizeByteSlice(candidate.Collatral)
	if err != nil {
		return nil, err
	}

	votePower, err := util.NewUint128FromFixedSizeByteSlice(candidate.VotePower)
	if err != nil {
		return nil, err
	}

	return &rpcpb.Candidate{
		Address:   byteutils.Bytes2Hex(candidate.Address),
		Collatral: collatral.String(),
		VotePower: votePower.String(),
	}, nil
}

func coreTx2rpcTx(tx *core.Transaction, executed bool) *rpcpb.GetTransactionResponse {
	return &rpcpb.GetTransactionResponse{
		Hash:      byteutils.Bytes2Hex(tx.Hash()),
		From:      tx.From().Hex(),
		To:        tx.To().Hex(),
		Value:     tx.Value().String(),
		Timestamp: tx.Timestamp(),
		Data: &rpcpb.TransactionData{
			Type:    tx.Type(),
			Payload: byteutils.Bytes2Hex(tx.Payload()),
		},
		Nonce:     tx.Nonce(),
		ChainId:   tx.ChainID(),
		Alg:       uint32(tx.Alg()),
		Sign:      byteutils.Bytes2Hex(tx.Sign()),
		PayerSign: byteutils.Bytes2Hex(tx.PayerSign()),
		Executed:  executed,
	}
}

func coreTxs2rpcTxs(txs core.Transactions, executed bool) []*rpcpb.GetTransactionResponse {
	var rpcTxs []*rpcpb.GetTransactionResponse
	for _, tx := range txs {
		rpcTx := coreTx2rpcTx(tx, executed)
		rpcTxs = append(rpcTxs, rpcTx)
	}
	return rpcTxs
}

func rpcPayload2payloadBuffer(txData *rpcpb.TransactionData) ([]byte, error) {
	var addRecord *core.AddRecordPayload
	var addCertification *core.AddCertificationPayload
	var revokeCertification *core.RevokeCertificationPayload

	switch txData.Type {
	case core.TxOpAddRecord:
		json.Unmarshal([]byte(txData.Payload), &addRecord)
		payload := core.NewAddRecordPayload(addRecord.Hash)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOpAddCertification:
		json.Unmarshal([]byte(txData.Payload), &addCertification)
		payload := core.NewAddCertificationPayload(addCertification.IssueTime,
			addCertification.ExpirationTime, addCertification.CertificateHash)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOpRevokeCertification:
		json.Unmarshal([]byte(txData.Payload), &revokeCertification)
		payload := core.NewRevokeCertificationPayload(revokeCertification.CertificateHash)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOpTransfer, core.TxOpVest, core.TxOpWithdrawVesting, dpos.TxOpBecomeCandidate, dpos.TxOpQuitCandidacy, dpos.TxOpVote:
		return nil, nil
	default:
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidDataType)
	}
}
