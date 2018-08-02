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
	"regexp"

	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func coreAcc2rpcAcc(acc *core.Account) (*rpcpb.GetAccountStateResponse, error) {
	var txsFrom []string
	var txsTo []string
	for _, hash := range acc.TxsFrom() {
		txsFrom = append(txsFrom, byteutils.Bytes2Hex(hash))
	}
	for _, hash := range acc.TxsTo() {
		txsTo = append(txsTo, byteutils.Bytes2Hex(hash))
	}
	return &rpcpb.GetAccountStateResponse{
		Address:       byteutils.Bytes2Hex(acc.Address()),
		Balance:       acc.Balance().String(),
		CertsIssued:   nil, // TODO @ggomma
		CertsReceived: nil, // TODO @ggomma
		Nonce:         acc.Nonce(),
		Records:       nil, // TODO @ggomma
		Vesting:       acc.Vesting().String(),
		Voted:         byteutils.Bytes2Hex(acc.Voted()),
		TxsSend:       txsFrom,
		TxsGet:        txsTo,
	}, nil
}

func corePbTx2rpcPbTx(pbTx *corepb.Transaction, executed bool) (*rpcpb.TransactionResponse, error) {
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
		Nonce:     pbTx.Nonce,
		ChainId:   pbTx.ChainId,
		Alg:       pbTx.Alg,
		Sign:      byteutils.Bytes2Hex(pbTx.Sign),
		PayerSign: byteutils.Bytes2Hex(pbTx.PayerSign),
		Executed:  executed,
	}, nil
}

func corePbBlock2rpcPbBlock(pbBlock *corepb.Block) (*rpcpb.BlockResponse, error) {
	var rpcPbTxs []*rpcpb.TransactionResponse
	for _, pbTx := range pbBlock.GetTransactions() {
		rpcPbTx, err := corePbTx2rpcPbTx(pbTx, true)
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
		ConsensusRoot: byteutils.Bytes2Hex(pbBlock.Header.DposRoot),
		Transactions:  rpcPbTxs,
		Height:        pbBlock.Height,
	}, nil
}

func generatePayloadBuf(txData *rpcpb.TransactionData) ([]byte, error) {
	var addRecord *core.AddRecordPayload
	var addCertification *core.AddCertificationPayload
	var revokeCertification *core.RevokeCertificationPayload

	switch txData.Type {
	case core.TxOpTransfer:
		return nil, nil
	case core.TxOpAddRecord:
		json.Unmarshal([]byte(txData.Payload), &addRecord)
		payload := core.NewAddRecordPayload(addRecord.Hash)
		payloadBuf, err := payload.ToBytes()
		if err != nil {
			return nil, err
		}
		return payloadBuf, nil
	case core.TxOpVest:
		return nil, nil
	case core.TxOpWithdrawVesting:
		return nil, nil
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
	case dpos.TxOpBecomeCandidate:
		return nil, nil
	case dpos.TxOpQuitCandidacy:
		return nil, nil
	case dpos.TxOpVote:
		return nil, nil
	}
	return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidDataType)
}

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

// GetMedState return mednet state
func (s *APIService) GetMedState(ctx context.Context, req *rpcpb.NonParamsRequest) (*rpcpb.GetMedStateResponse, error) {
	tailBlock := s.bm.TailBlock()
	lib := s.bm.LIB()

	return &rpcpb.GetMedStateResponse{
		ChainId: tailBlock.ChainID(),
		Tail:    byteutils.Bytes2Hex(tailBlock.Hash()),
		Height:  tailBlock.Height(),
		LIB:     byteutils.Bytes2Hex(lib.Hash()),
	}, nil
}

// GetAccountState handles GetAccountState rpc.
func (s *APIService) GetAccountState(ctx context.Context, req *rpcpb.GetAccountStateRequest) (*rpcpb.GetAccountStateResponse, error) {
	var block *core.Block
	var err error
	switch req.Height {
	case GENESIS:
		block, err = s.bm.BlockByHeight(1)
	case CONFIRMED:
		block = s.bm.LIB()
	case TAIL:
		block = s.bm.TailBlock()
	default:
		height, err := strconv.ParseUint(req.Height, 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, ErrMsgConvertBlockHeightFailed)
		}
		block, err = s.bm.BlockByHeight(height)
	}
	if block == nil || err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidBlockHeight)
	}
	acc, err := block.State().GetAccount(common.HexToAddress(req.Address))
	if err != nil {
		return &rpcpb.GetAccountStateResponse{
			Address:       req.Address,
			Balance:       util.Uint128Zero().String(),
			CertsIssued:   []string{},
			CertsReceived: []string{},
			Nonce:         0,
			Records:       []string{},
			Vesting:       util.Uint128Zero().String(),
			Voted:         "",
			TxsSend:       []string{},
			TxsGet:        []string{},
		}, nil
	}
	return &rpcpb.GetAccountStateResponse{
		Address:       byteutils.Bytes2Hex(acc.Address()),
		Balance:       acc.Balance().String(),
		CertsIssued:   byteutils.BytesSlice2HexSlice(acc.CertsIssued()),
		CertsReceived: byteutils.BytesSlice2HexSlice(acc.CertsReceived()),
		Nonce:         acc.Nonce(),
		Records:       byteutils.BytesSlice2HexSlice(acc.Records()),
		Vesting:       acc.Vesting().String(),
		Voted:         byteutils.Bytes2Hex(acc.Voted()),
		TxsSend:       byteutils.BytesSlice2HexSlice(acc.TxsFrom()),
		TxsGet:        byteutils.BytesSlice2HexSlice(acc.TxsTo()),
	}, nil
}

// GetAccounts return all account in the blockchain
func (s *APIService) GetAccounts(ctx context.Context, req *rpcpb.NonParamsRequest) (*rpcpb.AccountsResponse, error) {
	var rpcAccs []*rpcpb.GetAccountStateResponse

	block := s.bm.TailBlock()

	accs, err := block.State().GetAccounts()
	if err != nil {
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}
	for _, acc := range accs {
		rpcAcc, err := coreAcc2rpcAcc(acc)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgConvertAccountFailed)
		}
		rpcAccs = append(rpcAccs, rpcAcc)
	}

	return &rpcpb.AccountsResponse{
		Accounts: rpcAccs,
	}, nil
}

// GetBlock returns block
func (s *APIService) GetBlock(ctx context.Context, req *rpcpb.GetBlockRequest) (*rpcpb.BlockResponse, error) {
	var block *core.Block
	var err error
	switch req.Hash {
	case GENESIS:
		block, err = s.bm.BlockByHeight(1)
	case CONFIRMED:
		block = s.bm.LIB()
	case TAIL:
		block = s.bm.TailBlock()
	default:
		if number, _ := regexp.MatchString("^[0-9]*$", req.Hash); number {
			height, err := strconv.ParseUint(req.Hash, 10, 64)
			if height == 0 || err != nil {
				return nil, status.Error(codes.Internal, ErrMsgBlockNotFound)
			}
			block, err = s.bm.BlockByHeight(height)
			if err != nil {
				return nil, status.Error(codes.Internal, ErrMsgBlockNotFound)
			}
		} else {
			block = s.bm.BlockByHash(byteutils.FromHex(req.Hash))
		}
	}
	if block == nil || err != nil {
		return nil, status.Error(codes.NotFound, ErrMsgBlockNotFound)
	}
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

// GetBlocks returns blocks
func (s *APIService) GetBlocks(ctx context.Context, req *rpcpb.GetBlocksRequest) (*rpcpb.BlocksResponse, error) {
	var block *core.Block
	var rpcPbBlocks []*rpcpb.BlockResponse
	var err error

	if req.From > req.To {
		return nil, status.Error(codes.Internal, ErrMsgBlockNotFound)
	}

	for i := req.From; i <= req.To; i++ {
		block, err = s.bm.BlockByHeight(i)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgBlockNotFound)
		}

		pb, err := block.ToProto()
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgConvertBlockFailed)
		}

		if pbBlock, ok := pb.(*corepb.Block); ok {
			rpcPbBlock, err := corePbBlock2rpcPbBlock(pbBlock)
			if err != nil {
				return nil, status.Error(codes.Internal, ErrMsgConvertBlockFailed)
			}
			rpcPbBlocks = append(rpcPbBlocks, rpcPbBlock)
		} else {
			return nil, status.Error(codes.Internal, ErrMsgConvertBlockFailed)
		}
	}

	return &rpcpb.BlocksResponse{
		Blocks: rpcPbBlocks,
	}, nil
}

// GetCurrentAccountTransactions returns transactions of the account
func (s *APIService) GetCurrentAccountTransactions(ctx context.Context,
	req *rpcpb.GetCurrentAccountTransactionsRequest) (*rpcpb.TransactionsResponse, error) {
	var txs []*rpcpb.TransactionResponse

	address := common.HexToAddress(req.Address)
	poolTxs := s.tm.GetByAddress(address)
	tailBlock := s.bm.TailBlock()

	for _, tx := range poolTxs {
		corePbTx, err := tx.ToProto()
		if err != nil {
			continue
		}
		rpcPbTx, err := corePbTx2rpcPbTx(corePbTx.(*corepb.Transaction), false)
		if err != nil {
			continue
		}
		txs = append(txs, rpcPbTx)
		// Add send transaction twice if the address of from is as same as the address of to
		if tx.Type() == core.TxOpTransfer && tx.From() == tx.To() {
			txs = append(txs, rpcPbTx)
		}
	}

	if tailBlock != nil {
		acc, err := tailBlock.State().GetAccount(address)
		if err == nil {
			txList := append(acc.TxsTo(), acc.TxsFrom()...)
			for _, hash := range txList {
				pbTx := new(corepb.Transaction)
				pb, err := tailBlock.State().GetTx(hash)
				if err != nil {
					continue
				}
				err = proto.Unmarshal(pb, pbTx)
				if err != nil {
					continue
				}
				rpcPbTx, err := corePbTx2rpcPbTx(pbTx, true)
				if err != nil {
					continue
				}
				txs = append(txs, rpcPbTx)
			}
		}
	}

	return &rpcpb.TransactionsResponse{
		Transactions: txs,
	}, nil
}

// GetTransaction returns transaction
func (s *APIService) GetTransaction(ctx context.Context, req *rpcpb.GetTransactionRequest) (*rpcpb.TransactionResponse, error) {
	if len(req.Hash) != 64 {
		return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
	}
	executed := true

	tailBlock := s.bm.TailBlock()
	if tailBlock == nil {
		return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
	}

	txHash := byteutils.Hex2Bytes(req.Hash)

	pbTx := new(corepb.Transaction)
	pb, err := tailBlock.State().GetTx(txHash)
	if err != nil {
		if err != trie.ErrNotFound {
			return nil, status.Error(codes.Internal, ErrMsgGetTransactionFailed)
		}
		tx := s.tm.Get(txHash)
		if tx == nil {
			return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
		}
		prePbTx, err := tx.ToProto()
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgGetTransactionFailed)
		}
		pbTx = prePbTx.(*corepb.Transaction)
		executed = false
	} else {
		err = proto.Unmarshal(pb, pbTx)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgUnmarshalTransactionFailed)
		}
	}

	res, err := corePbTx2rpcPbTx(pbTx, executed)
	if err != nil {
		return nil, status.Error(codes.Internal, ErrMsgConvertTxResponseFailed)
	}
	return res, nil
}

// GetPendingTransactions sends all transactions in the transaction pool
func (s *APIService) GetPendingTransactions(ctx context.Context, req *rpcpb.NonParamsRequest) (*rpcpb.TransactionsResponse, error) {
	var rpcPbTxs []*rpcpb.TransactionResponse

	txs := s.tm.GetAll()
	for _, tx := range txs {
		corePbTx, err := tx.ToProto()
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgGetTransactionFailed)
		}
		rpcPbTx, err := corePbTx2rpcPbTx(corePbTx.(*corepb.Transaction), false)
		if err != nil {
			return nil, status.Error(codes.Internal, ErrMsgConvertTxResponseFailed)
		}
		rpcPbTxs = append(rpcPbTxs, rpcPbTx)
	}
	return &rpcpb.TransactionsResponse{
		Transactions: rpcPbTxs,
	}, nil
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
