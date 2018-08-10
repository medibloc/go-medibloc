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
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// APIService is blockchain api rpc service.
type APIService struct {
	bm *core.BlockManager
	tm *core.TransactionManager
	ee *core.EventEmitter
}

func newAPIService(bm *core.BlockManager, tm *core.TransactionManager, ee *core.EventEmitter) *APIService {
	return &APIService{
		bm: bm,
		tm: tm,
		ee: ee,
	}
}

// GetAccount handles GetAccount rpc.
func (s *APIService) GetAccount(ctx context.Context, req *rpcpb.GetAccountRequest) (*rpcpb.GetAccountResponse, error) {
	var block *core.Block
	var err error

	switch req.Type {
	case GENESIS:
		block, err = s.bm.BlockByHeight(1)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
		}
	case CONFIRMED:
		block = s.bm.LIB()
	case TAIL:
		block = s.bm.TailBlock()
	default:
		block, err = s.bm.BlockByHeight(req.Height)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidBlockHeight)
		}
	}
	if block == nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
	}

	acc, err := block.State().GetAccount(common.HexToAddress(req.Address))
	if err != nil {
		return coreAccount2rpcAccount(nil, req.Address), nil
	}
	return coreAccount2rpcAccount(acc, nil), nil
}

// GetBlock returns block
func (s *APIService) GetBlock(ctx context.Context, req *rpcpb.GetBlockRequest) (*rpcpb.GetBlockResponse, error) {
	var block *core.Block
	var err error

	if req.Hash != nil {
		block = s.bm.BlockByHash(byteutils.FromHex(req.Hash))
		if block == nil {
			return nil, status.Error(codes.Internal, ErrMsgBlockNotFound)
		}
	}

	switch req.Hash {
	case GENESIS:
		block, err = s.bm.BlockByHeight(1)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
		}
	case CONFIRMED:
		block = s.bm.LIB()
	case TAIL:
		block = s.bm.TailBlock()
	default:
		block, err = s.bm.BlockByHeight(req.Height)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidBlockHeight)
		}
	}
	if block == nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
	}

	return coreBlock2rpcBlock(block)
}

// GetCandidates returns all candidates
func (s *APIService) GetCandidates(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.GetCandidatesResponse, error) {
	var rpcCandidates []*rpcpb.Candidate
	block := s.bm.TailBlock()

	candidates, err := block.State().GetCandidates()
	if err != nil {
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}
	for _, candidate := range candidates {
		rpcCandidate, err := dposCandidate2rpcCandidate(candidate)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgConvertAccountFailed)
		}
		rpcCandidates = append(rpcCandidates, rpcCandidate)
	}
	return &rpcpb.CandidatesResponse{
		Candidates: rpcCandidates,
	}, nil
}

// GetDynasty returns all dynasty accounts
func (s *APIService) GetDynasty(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.GetDynastyResponse, error) {
	var addresses []string

	block := s.bm.TailBlock()

	addrs, err := block.State().GetDynasty()
	if err != nil {
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}
	for _, addr := range addrs {
		addresses = append(addresses, byteutils.Bytes2Hex(addr))
	}

	return &rpcpb.GetDynastyResponse{
		Addresses: addresses,
	}, nil
}

// GetMedState return mednet state
func (s *APIService) GetMedState(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.GetMedStateResponse, error) {
	tailBlock := s.bm.TailBlock()
	lib := s.bm.LIB()

	return &rpcpb.GetMedStateResponse{
		ChainId: tailBlock.ChainID(),
		Tail:    byteutils.Bytes2Hex(tailBlock.Hash()),
		Height:  tailBlock.Height(),
		LIB:     byteutils.Bytes2Hex(lib.Hash()),
	}, nil
}

// GetPendingTransactions sends all transactions in the transaction pool
func (s *APIService) GetPendingTransactions(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.GetTransactionsResponse, error) {
	txs := Transactions(s.tm.GetAll())
	rpcTxs, err := coreTxs2rpcTxs(txs)
	if err != nil {
		return nil, err
	}

	return &rpcpb.TransactionsResponse{
		Transactions: rpcTxs,
	}, nil
}

// GetTransaction returns transaction
func (s *APIService) GetTransaction(ctx context.Context, req *rpcpb.GetTransactionRequest) (*rpcpb.GetTransactionResponse, error) {
	if len(req.Hash) != 64 {
		return nil, status.Error(codes.NotFound, ErrMsgInvalidTxHash)
	}

	tailBlock := s.bm.TailBlock()
	if tailBlock == nil {
		return nil, status.Error(codes.NotFound, ErrMsgInternalError)
	}

	executed := true
	tx := new(core.Transaction)
	txHash := byteutils.Hex2Bytes(req.Hash)
	pb, err := tailBlock.State().GetTx(txHash)
	// If tx is in txPool
	if err != nil {
		if err != trie.ErrNotFound {
			return nil, status.Error(codes.Internal, ErrMsgGetTransactionFailed)
		}
		// Get tx from txPool (type *Transaction)
		tx = s.tm.Get(txHash)
		if tx == nil {
			return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
		}
		executed = false
	} else { // If tx is already included in a block
		err := tx.FromProto(pb)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgUnmarshalTransactionFailed)
		}
	}

	return coreTx2rpcTx(tx, executed), nil
}

// GetAccountTransactions returns transactions of the account
func (s *APIService) GetAccountTransactions(ctx context.Context,
	req *rpcpb.GetAccountTransactionsRequest) (*rpcpb.GetTransactionsResponse, error) {
	var txs []*rpcpb.GetTransactionResponse

	address := common.HexToAddress(req.Address)
	poolTxs := s.tm.GetByAddress(address)
	for _, tx := range poolTxs {
		txs = append(txs, coreTx2rpcTx(tx, false))
		// Add send transaction twice if the address of from is as same as the address of to
		if tx.Type() == core.TxOpTransfer && tx.From() == tx.To() {
			txs = append(txs, coreTx2RpcTx(tx, false))
		}
	}

	tailBlock := s.bm.TailBlock()
	if tailBlock == nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
	}

	acc, err := tailBlock.State().GetAccount(address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
	}
	if acc != nil {
		txList := append(acc.TxsTo(), acc.TxsFrom()...)
		for _, hash := range txList {
			tx := new(core.Transaction)
			pb, err := tailBlock.State().GetTx(hash)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
			}
			err = tx.FromProto(pb)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, ErrMsgInternalError)
			}
			txs = append(txs, coreTx2rpcTx(tx))
		}
	}

	return &rpcpb.GetTransactionsResponse{
		Transactions: txs,
	}, nil
}

// SendTransaction sends transaction
func (s *APIService) SendTransaction(ctx context.Context, req *rpcpb.SendTransactionRequest) (*rpcpb.SendTransactionResponse, error) {
	value, err := util.NewUint128FromString(req.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTxValue)
	}
	payloadBuf, err := rpcPayload2payloadBuffer(req.Data)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTxDataPayload)
	}
	tx, err := core.BuildTransaction(
		req.ChainId,
		common.HexToAddress(req.From),
		common.HexToAddress(req.To),
		value,
		req.Nonce,
		req.Timestamp,
		&corepb.Data{
			Type:    req.Data.Type,
			Payload: payloadBuf,
		},
		byteutils.Hex2Bytes(req.Hash),
		req.Alg,
		byteutils.Hex2Bytes(req.Sign),
		byteutils.Hex2Bytes(req.PayerSign))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgBuildTransactionFail)
	}
	if err = s.tm.PushAndRelay(tx); err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTransaction)
	}
	return &rpcpb.SendTransactionResponse{
		Hash: byteutils.Bytes2Hex(tx.Hash()),
	}, nil
}

// Subscribe to listen event
func (s *APIService) Subscribe(req *rpcpb.SubscribeRequest, stream rpcpb.ApiService_SubscribeServer) error {

	eventSub, err := core.NewEventSubscriber(1024, req.Topics)
	if err != nil {
		return err
	}

	s.ee.Register(eventSub)
	defer s.ee.Deregister(eventSub)

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event := <-eventSub.EventChan():
			err := stream.Send(&rpcpb.SubscribeResponse{
				Topic: event.Topic,
				Data:  event.Data,
			})
			// TODO : Send timeout
			if err != nil {
				return err
			}
		}
	}
}
