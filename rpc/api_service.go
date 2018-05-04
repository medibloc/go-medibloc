package rpc

import (
	"encoding/hex"

	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/net/context"
)

// APIService errors
var (
	ErrTransctionNotFound = errors.New("transaction not found")
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

// GetTransaction returns transaction
func (s *APIService) GetTransaction(ctx context.Context, req *rpcpb.GetTransactionRequest) (*rpcpb.TransactionResponse, error) {
	tailBlock := s.bm.TailBlock()
	if tailBlock != nil {
		// TODO: check req.Hash is nil
		pb, err := tailBlock.State().GetTx(common.HexToHash(req.Hash))
		if err != nil {
			return nil, ErrTransctionNotFound
		}
		pbTx := new(corepb.Transaction)
		proto.Unmarshal(pb, pbTx)

		var data *rpcpb.TransactionData
		// TODO: add more transaction
		if pbTx.Data.Type == core.TxPayloadBinaryType {
			data = &rpcpb.TransactionData{
				Type:    pbTx.Data.Type,
				Payload: nil,
			}
		} else if pbTx.Data.Type == core.TxOperationRegisterWKey {
			txPayload, err := core.BytesToRegisterWriterPayload(pbTx.Data.Payload)
			if err != nil {
				return nil, err
			}
			data = &rpcpb.TransactionData{
				Type: pbTx.Data.Type,
				Payload: &rpcpb.Payload{
					Writer: txPayload.Writer.String(),
				},
			}
		}

		return &rpcpb.TransactionResponse{
			Hash:      byteutils.Bytes2Hex(pbTx.Hash),
			From:      byteutils.Bytes2Hex(pbTx.From),
			To:        byteutils.Bytes2Hex(pbTx.To),
			Value:     byteutils.Bytes2Hex(pbTx.From),
			Timestamp: pbTx.Timestamp,
			Data:      data,
			Nonce:     pbTx.Nonce,
			ChainId:   pbTx.ChainId,
			Alg:       pbTx.Alg,
			Sign:      byteutils.Bytes2Hex(pbTx.Sign),
		}, nil
	}
	return &rpcpb.TransactionResponse{}, nil
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
	var data = &corepb.Data{
		Type:    "",
		Payload: nil,
	}
	// TODO: add more transaction
	if req.Data.Type == core.TxPayloadBinaryType {
		data = &corepb.Data{
			Type:    req.Data.Type,
			Payload: nil,
		}
	} else {
		data = &corepb.Data{
			Type:    req.Data.Type,
			Payload: nil,
		}
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
