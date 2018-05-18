package rpc

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
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
		Tail:    tailBlock.Hash().Hex(),
	}, nil
}

// GetAccountState handles GetAccountState rpc.
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
