package rpc

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"github.com/medibloc/go-medibloc/util"
	"golang.org/x/net/context"
)

// AdminService is admin api rpc service.
type AdminService struct {
	bridge Bridge
	server GRPCServer
}

// SendTransaction handles SendTransaction rpc.
func (s *AdminService) SendTransaction(ctx context.Context, req *rpcpb.TransactionRequest) (*rpcpb.TransactionResponse, error) {
	bm := s.bridge.BlockManager()
	value, err := util.NewUint128FromString(req.Value)
	if err != nil {
		return nil, err
	}
	tx, err := core.NewTransactionWithSign(
		bm.ChainID(),
		common.HexToAddress(req.From),
		common.HexToAddress(req.To),
		value,
		req.Nonce,
		core.TxPayloadBinaryType,
		nil,
		common.BytesToHash(req.Hash),
		req.Sign)

	if err != nil {
		return nil, err
	}
	if err = s.bridge.TransactionManager().Push(tx); err != nil {
		return nil, err
	}
	return &rpcpb.TransactionResponse{
		Txhash: tx.Hash().Str(),
	}, nil
}
