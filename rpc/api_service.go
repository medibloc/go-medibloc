package rpc

import (
	"github.com/medibloc/go-medibloc/common"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"golang.org/x/net/context"
)

//
// FAKE DATA BEGIN
//

type account struct {
	address string
	balance float32
	nonce   uint64
	ttype   uint32
}

func makeFakeBlockChain() map[uint64][]*account {
	blockChain := map[uint64][]*account{
		1: []*account{
			&account{
				address: "M00001",
				balance: 1,
				nonce:   2,
				ttype:   1,
			},
			&account{
				address: "M00002",
				balance: 1,
				nonce:   2,
				ttype:   1,
			},
		},
		2: []*account{
			&account{
				address: "M00001",
				balance: 5,
				nonce:   6,
				ttype:   1,
			},
			&account{
				address: "M00002",
				balance: 7,
				nonce:   12,
				ttype:   1,
			},
		},
	}
	return blockChain
}

//
// FAKE DATA ENDS
//

// APIService is blockchain api rpc service.
type APIService struct {
	bridge Bridge
	server GRPCServer
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
	bm := s.bridge.BlockManager()
	tailBlock := bm.TailBlock()
	return &rpcpb.GetMedStateResponse{
		ChainId: tailBlock.ChainID(),
		Height:  tailBlock.Height(),
		Tail:    tailBlock.Hash().Str(),
	}, nil
}

// GetAccountState handles GetAccountState rpc.
func (s *APIService) GetAccountState(ctx context.Context, req *rpcpb.GetAccountStateRequest) (*rpcpb.GetAccountStateResponse, error) {
	bm := s.bridge.BlockManager()
	tailBlock := bm.TailBlock()
	acc, err := tailBlock.State().GetAccount(common.HexToAddress(req.Address))
	if err != nil {
		return nil, err
	}
	// height := req.Height TODO get state for height
	return &rpcpb.GetAccountStateResponse{
		Balance: acc.Balance().String(),
		Nonce:   acc.Nonce(),
	}, nil
}
