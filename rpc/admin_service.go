package rpc

import (
	"encoding/hex"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"github.com/medibloc/go-medibloc/util"
	"golang.org/x/net/context"
)

// AdminService is admin api rpc service.
type AdminService struct {
	bm *core.BlockManager
	tm *core.TransactionManager
}

func newAdminService(bm *core.BlockManager, tm *core.TransactionManager) *AdminService {
	return &AdminService{
		bm: bm,
		tm: tm,
	}
}

// SendTransaction handles SendTransaction rpc.
func (s *AdminService) SendTransaction(ctx context.Context, req *rpcpb.TransactionRequest) (*rpcpb.TransactionResponse, error) {
	value, err := util.NewUint128FromString(req.Value)
	if err != nil {
		return nil, err
	}
	sign, err := hex.DecodeString(req.Sign)
	if err != nil {
		return nil, err
	}
	tx, err := core.NewTransactionWithSign(
		s.bm.ChainID(),
		common.HexToAddress(req.From),
		common.HexToAddress(req.To),
		value,
		req.Nonce,
		core.TxPayloadBinaryType,
		nil,
		common.HexToHash(req.Hash),
		sign)

	if err != nil {
		return nil, err
	}
	if err = s.tm.Push(tx); err != nil {
		return nil, err
	}
	return &rpcpb.TransactionResponse{
		Txhash: tx.Hash().Hex(),
	}, nil
}
