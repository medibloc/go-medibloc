package rpc

import (
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
	server GRPCServer
}

// GetAccountState handles GetAccountState rpc.
func (s *APIService) GetAccountState(ctx context.Context, req *rpcpb.GetAccountStateRequest) (*rpcpb.GetAccountStateResponse, error) {
	addr := req.Address
	height := req.Height
	blockChain := makeFakeBlockChain()
	for _, acc := range blockChain[height] {
		if acc.address == addr {
			return &rpcpb.GetAccountStateResponse{
				Balance: acc.balance,
				Nonce:   acc.nonce,
				Type:    acc.ttype,
			}, nil
		}
	}
	return nil, nil
}
