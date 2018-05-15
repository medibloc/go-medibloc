package rpc

import (
	"encoding/json"

	"github.com/gogo/protobuf/proto"
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

// APIService errors
var (
	ErrMsgAccountNotFound            = "account not found"
	ErrMsgBlockNotFound              = "block not found"
	ErrMsgBuildTransactionFail       = "cannot build transaction"
	ErrMsgConvertBlockFailed         = "cannot convert block"
	ErrMsgConvertBlockResponseFailed = "cannot convert block response"
	ErrMsgConvertTxResponseFailed    = "cannot convert transaction response"
	ErrMsgGetTransactionFailed       = "cannot get transaction from state"
	ErrMsgInvalidDataType            = "invalid transaction data type"
	ErrMsgInvalidTransaction         = "invalid transaction"
	ErrMsgInvalidTxValue             = "invalid transaction value"
	ErrMsgInvalidTxDataPayload       = "invalid transaction data payload"
	ErrMsgTransactionNotFound        = "transaction not found"
	ErrMsgUnmarshalTransactionFailed = "cannot unmarshal transaction"
)

func corePbTx2rpcPbTx(pbTx *corepb.Transaction) (*rpcpb.TransactionResponse, error) {
	value, err := util.NewUint128FromFixedSizeByteSlice(pbTx.Value)
	if err != nil {
		return nil, err
	}
	return &rpcpb.TransactionResponse{
		Hash:      byteutils.Bytes2Hex(pbTx.Hash),
		From:      byteutils.Bytes2Hex(pbTx.From),
		To:        byteutils.Bytes2Hex(pbTx.To),
		Value:     value.String(),
		Timestamp: pbTx.Timestamp,
		Data: &rpcpb.TransactionData{
			Type:    pbTx.Data.Type,
			Payload: string(pbTx.Data.Payload),
		},
		Nonce:   pbTx.Nonce,
		ChainId: pbTx.ChainId,
		Alg:     pbTx.Alg,
		Sign:    byteutils.Bytes2Hex(pbTx.Sign),
	}, nil
}

func corePbBlock2rpcPbBlock(pbBlock *corepb.Block) (*rpcpb.BlockResponse, error) {
	var rpcPbTxs []*rpcpb.TransactionResponse
	for _, pbTx := range pbBlock.GetTransactions() {
		rpcPbTx, err := corePbTx2rpcPbTx(pbTx)
		if err != nil {
			return nil, err
		}
		rpcPbTxs = append(rpcPbTxs, rpcPbTx)
	}

	return &rpcpb.BlockResponse{
		Hash:          byteutils.Bytes2Hex(pbBlock.Header.Hash),
		ParentHash:    byteutils.Bytes2Hex(pbBlock.Header.ParentHash),
		Coinbase:      byteutils.Bytes2Hex(pbBlock.Header.Coinbase),
		Timestamp:     pbBlock.Header.Timestamp,
		ChainId:       pbBlock.Header.ChainId,
		Alg:           pbBlock.Header.Alg,
		Sign:          byteutils.Bytes2Hex(pbBlock.Header.Sign),
		AccsRoot:      byteutils.Bytes2Hex(pbBlock.Header.AccsRoot),
		TxsRoot:       byteutils.Bytes2Hex(pbBlock.Header.TxsRoot),
		UsageRoot:     byteutils.Bytes2Hex(pbBlock.Header.UsageRoot),
		RecordsRoot:   byteutils.Bytes2Hex(pbBlock.Header.RecordsRoot),
		ConsensusRoot: byteutils.Bytes2Hex(pbBlock.Header.ConsensusRoot),
		Transactions:  rpcPbTxs,
		Height:        pbBlock.Height,
	}, nil
}

func generatePayloadBuf(txData *rpcpb.TransactionData) ([]byte, error) {
	var registerWriter *core.RegisterWriterPayload
	var addRecord *core.AddRecordPayload
	// var removeWriter *core.RemoveWriterPayload
	// var addRecordReader *core.AddRecordReaderPayload

	switch txData.Type {
	case core.TxOperationSend:
	case core.TxOperationAddRecord:
		json.Unmarshal([]byte(txData.Payload), &addRecord)
		payload := core.NewAddRecordPayload(addRecord.Hash, addRecord.Storage, addRecord.EncKey, addRecord.Seed)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOperationAddRecordReader:
	case core.TxOperationVest:
	case core.TxOperationWithdrawVesting:
	case core.TxOperationRegisterWKey:
		json.Unmarshal([]byte(txData.Payload), &registerWriter)
		payload := core.NewRegisterWriterPayload(registerWriter.Writer)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOperationRemoveWKey:
	case core.TxPayloadBinaryType:
		return nil, nil
	}
	return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidDataType)
}

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
		Tail:    tailBlock.Hash().String(),
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
		return nil, status.Error(codes.NotFound, ErrMsgAccountNotFound)
	}
	return &rpcpb.GetAccountStateResponse{
		Balance: acc.Balance().String(),
		Nonce:   acc.Nonce(),
	}, nil
}

// GetBlock returns block
func (s *APIService) GetBlock(ctx context.Context, req *rpcpb.GetBlockRequest) (*rpcpb.BlockResponse, error) {
	block := s.bm.BlockByHash(common.HexToHash(req.Hash))
	if block != nil {
		pb, err := block.ToProto()
		if err == nil {
			if pbBlock, ok := pb.(*corepb.Block); ok {
				res, err := corePbBlock2rpcPbBlock(pbBlock)
				if err != nil {
					return nil, status.Error(codes.Internal, ErrMsgConvertBlockResponseFailed)
				}
				return res, nil
			}
		}
		return nil, status.Error(codes.Internal, ErrMsgConvertBlockFailed)
	}
	return nil, status.Error(codes.NotFound, ErrMsgBlockNotFound)
}

// GetTailBlock returns tail block
func (s *APIService) GetTailBlock(ctx context.Context, req *rpcpb.NonParamsRequest) (*rpcpb.BlockResponse, error) {
	tailBlock := s.bm.TailBlock()
	if tailBlock != nil {
		pb, err := tailBlock.ToProto()
		if err == nil {
			if pbBlock, ok := pb.(*corepb.Block); ok {
				res, err := corePbBlock2rpcPbBlock(pbBlock)
				if err != nil {
					return nil, status.Error(codes.Internal, ErrMsgConvertBlockResponseFailed)
				}
				return res, nil
			}
		}
		return nil, status.Error(codes.Internal, ErrMsgConvertBlockFailed)
	}
	return nil, status.Error(codes.NotFound, ErrMsgBlockNotFound)
}

// GetTransaction returns transaction
func (s *APIService) GetTransaction(ctx context.Context, req *rpcpb.GetTransactionRequest) (*rpcpb.TransactionResponse, error) {
	tailBlock := s.bm.TailBlock()
	if tailBlock != nil {
		// TODO: check req.Hash is nil
		pb, err := tailBlock.State().GetTx(common.HexToHash(req.Hash))
		if err != nil {
			if err == trie.ErrNotFound {
				return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
			}
			return nil, status.Error(codes.Internal, ErrMsgGetTransactionFailed)
		}
		pbTx := new(corepb.Transaction)
		err = proto.Unmarshal(pb, pbTx)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgUnmarshalTransactionFailed)
		}
		res, err := corePbTx2rpcPbTx(pbTx)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgConvertTxResponseFailed)
		}
		return res, nil
	}
	return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
}

// SendTransaction sends transaction
func (s *APIService) SendTransaction(ctx context.Context, req *rpcpb.SendTransactionRequest) (*rpcpb.SendTransactionResponse, error) {
	value, err := util.NewUint128FromString(req.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTxValue)
	}
	payloadBuf, err := generatePayloadBuf(req.Data)
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
		common.HexToHash(req.Hash),
		req.Alg,
		byteutils.Hex2Bytes(req.Sign))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgBuildTransactionFail)
	}
	if err = s.tm.Push(tx); err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTransaction)
	}
	return &rpcpb.SendTransactionResponse{
		Hash: tx.Hash().Hex(),
	}, nil
}
