package rpc

import (
	"encoding/hex"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util"
	"golang.org/x/net/context"
)

// APIService is blockchain api rpc service.
type APIService struct {
	bm *core.BlockManager
	tm *core.TransactionManager
}

func newAPIService(bm *core.BlockManager, tm *core.TransactionManager) *APIService {
	return &APIService{
		bm: bm,
		tm: tm,
	}
}

// GetMedState return mednet state
// chain_id
// tail
// lib (TODO)
// height
// protocol_version (TODO)
// synchronized (TODO)
// version (TODO)
func (s *APIService) GetMedState(ctx context.Context, req *rpcpb.NonParamsRequest) (*rpcpb.GetMedStateResponse, error) {
	tailBlock := s.bm.TailBlock()
	return &rpcpb.GetMedStateResponse{
		ChainId: tailBlock.ChainID(),
		Height:  tailBlock.Height(),
		Tail:    tailBlock.Hash().Str(),
	}, nil
}

// GetAccountState handles GetAccountState rpc.
// balance
// nonce
// staking (TODO)
func (s *APIService) GetAccountState(ctx context.Context, req *rpcpb.GetAccountStateRequest) (*rpcpb.GetAccountStateResponse, error) {
	// height := req.Height TODO get state for height
	tailBlock := s.bm.TailBlock()
	acc, err := tailBlock.State().GetAccount(common.HexToAddress(req.Address))
	if err != nil {
		return nil, err
	}
	return &rpcpb.GetAccountStateResponse{
		Balance: acc.Balance().String(),
		Nonce:   acc.Nonce(),
	}, nil
}

// GetBlock returns block
func (s *APIService) GetBlock(ctx context.Context, req *rpcpb.GetBlockRequest) (*rpcpb.GetBlockResponse, error) {
	block := s.bm.BlockByHash(common.HexToHash(req.Hash))
	if block != nil {
		pb, err := block.ToProto()
		if err != nil {
			return nil, err
		}
		if pbBlock, ok := pb.(*corepb.Block); ok {
			return &rpcpb.GetBlockResponse{
				Block: pbBlock,
			}, nil
		}
	}
	return &rpcpb.GetBlockResponse{
		Block: nil,
	}, nil
}

// GetTailBlock returns tail block
func (s *APIService) GetTailBlock(ctx context.Context, req *rpcpb.NonParamsRequest) (*rpcpb.GetBlockResponse, error) {
	tailBlock := s.bm.TailBlock()
	if tailBlock != nil {
		pb, err := tailBlock.ToProto()
		if err != nil {
			return nil, err
		}
		if pbBlock, ok := pb.(*corepb.Block); ok {
			return &rpcpb.GetBlockResponse{
				Block: pbBlock,
			}, nil
		}
	}
	return &rpcpb.GetBlockResponse{
		Block: nil,
	}, nil
}

// SendTransaction sends transaction
func (s *APIService) SendTransaction(ctx context.Context, req *rpcpb.SendTransactionRequest) (*rpcpb.SendTransactionResponse, error) {
	value, err := util.NewUint128FromString(req.Value)
	if err != nil {
		return nil, err
	}
	sign, err := hex.DecodeString(req.Sign)
	if err != nil {
		return nil, err
	}
	data := &corepb.Data{
		Type: req.Data.Type,
		// TODO: convert payload
		Payload: nil,
	}
	tx, err := core.BuildTransaction(
		req.ChainId,
		common.HexToAddress(req.From),
		common.HexToAddress(req.To),
		value,
		req.Nonce,
		req.Timestamp,
		data,
		common.HexToHash(req.Hash),
		req.Alg,
		sign)
	if err != nil {
		return nil, err
	}
	if err = s.tm.Push(tx); err != nil {
		return nil, err
	}
	return &rpcpb.SendTransactionResponse{
		Hash: tx.Hash().Hex(),
	}, nil
}
