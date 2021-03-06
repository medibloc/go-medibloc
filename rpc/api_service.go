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
	"encoding/hex"

	"github.com/medibloc/go-medibloc/event"
	"github.com/sirupsen/logrus"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/net"
	rpcpb "github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/medibloc/go-medibloc/util/math"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// APIService is blockchain api rpc service.
type APIService struct {
	bm *core.BlockManager
	tm *core.TransactionManager
	ee *event.Emitter
	ns net.Service
}

func newAPIService(bm *core.BlockManager, tm *core.TransactionManager, ee *event.Emitter, ns net.Service) *APIService {
	return &APIService{
		bm: bm,
		tm: tm,
		ee: ee,
		ns: ns,
	}
}

// GetAccount handles GetAccount rpc.
func (s *APIService) GetAccount(ctx context.Context, req *rpcpb.GetAccountRequest) (*rpcpb.Account,
	error) {
	var block *core.Block
	var err error

	// XOR
	if !(math.XOR(!common.IsHexAddress(req.Address), req.Alias == "") && math.XOR(req.Type == "", req.Height == 0)) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidRequest)
	}

	if req.Type != "" {
		switch req.GetType() {
		case GENESIS:
			block, err = s.bm.BlockByHeight(core.GenesisHeight)
			if err != nil {
				logging.Console().WithFields(logrus.Fields{
					"err": err,
				}).Error("Could not get the genesis block.")
				return nil, status.Error(codes.Internal, ErrMsgInternalError)
			}
		case CONFIRMED:
			block = s.bm.LIB()
		case TAIL:
			block = s.bm.TailBlock()
		default:
			return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidBlockType)
		}
	} else {
		block, err = s.bm.BlockByHeight(req.GetHeight())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidBlockHeight)
		}
	}

	if block == nil {
		logging.Console().WithFields(logrus.Fields{
			"height": req.Height,
			"type":   req.Type,
		}).Error("Could not get the block.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	// TODO: api service change
	var addr string
	if common.IsHexAddress(req.Address) {
		addr = req.Address
	} else {
		acc, err := block.State().GetAccountByAlias(req.Alias)
		if err == trie.ErrNotFound {
			return nil, status.Error(codes.NotFound, ErrMsgAliasNotFound)
		}
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"alias": req.Alias,
				"err":   err,
			}).Error("Failed to get alias account from state.")
			return nil, status.Error(codes.Internal, ErrMsgInternalError)
		}
		addr = acc.Address.String()
	}

	addrAddr, err := common.HexToAddress(addr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidAddress)
	}
	acc, err := block.State().GetAccount(addrAddr)
	if err != nil && err != trie.ErrNotFound {
		logging.Console().WithFields(logrus.Fields{
			"addr": addrAddr,
			"err":  err,
		}).Error("Failed to get account from state.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}
	return coreAccount2rpcAccount(acc, block.Timestamp(), addr)
}

// GetBlock returns block
func (s *APIService) GetBlock(ctx context.Context, req *rpcpb.GetBlockRequest) (*rpcpb.Block, error) {
	var block *core.Block
	var err error

	if !math.TernaryXOR(common.IsHash(req.Hash), req.Type != "", req.Height != 0) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidRequest)
	}

	if common.IsHash(req.Hash) {
		hash, err := byteutils.FromHex(req.Hash)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidBlockHash)
		}
		block = s.bm.BlockByHash(hash)
		if block == nil {
			return nil, status.Error(codes.NotFound, ErrMsgBlockNotFound)
		}
		return coreBlock2rpcBlock(block, false), nil
	}

	switch req.Type {
	case GENESIS:
		block, err = s.bm.BlockByHeight(1)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Could not get the genesis block.")
			return nil, status.Error(codes.Internal, ErrMsgInternalError)
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
		logging.Console().WithFields(logrus.Fields{
			"hash":   req.Hash,
			"height": req.Height,
			"type":   req.Type,
		}).Error("Could not get the block.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	return coreBlock2rpcBlock(block, false), nil
}

// GetBlocks returns blocks
func (s *APIService) GetBlocks(ctx context.Context, req *rpcpb.GetBlocksRequest) (*rpcpb.Blocks, error) {
	var rpcBlocks []*rpcpb.Block

	if req.From > req.To || req.From < 1 {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidRequest)
	}

	if req.To-req.From > MaxBlocksCount {
		return nil, status.Error(codes.InvalidArgument, ErrMsgTooManyBlocksRequest)
	}

	if s.bm.TailBlock().Height() < req.To {
		req.To = s.bm.TailBlock().Height()
	}

	for i := req.From; i <= req.To; i++ {
		block, err := s.bm.BlockByHeight(i)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"height":         i,
				"mainTailHeight": s.bm.TailBlock().Height(),
			}).Error("Could not get the block.")
			return nil, status.Error(codes.Internal, ErrMsgInternalError)
		}

		rpcBlock := coreBlock2rpcBlock(block, true)
		rpcBlocks = append(rpcBlocks, rpcBlock)
	}

	return &rpcpb.Blocks{
		Blocks: rpcBlocks,
	}, nil
}

// GetCandidate returns matched candidate information
func (s *APIService) GetCandidate(ctx context.Context, req *rpcpb.GetCandidateRequest) (*rpcpb.Candidate, error) {
	block := s.bm.TailBlock()

	if len(req.CandidateId) != 64 {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidRequest)
	}

	cid, err := byteutils.Hex2Bytes(req.CandidateId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidCandidateID)
	}

	candidate, err := block.State().DposState().GetCandidate(cid)
	if err == trie.ErrNotFound {
		return nil, status.Error(codes.NotFound, ErrMsgCandidateNotFound)
	}
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"candidateID": cid,
			"err":         err,
		}).Error("Could not get the candidate.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	return dposCandidate2rpcCandidate(candidate), nil
}

// GetCandidates returns all candidates
func (s *APIService) GetCandidates(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.Candidates,
	error) {
	var rpcCandidates []*rpcpb.Candidate
	block := s.bm.TailBlock()

	candidates, err := block.State().DposState().GetCandidates()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get list of candidates from state.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	for _, candidate := range candidates {
		rpcCandidates = append(rpcCandidates, dposCandidate2rpcCandidate(candidate))
	}
	return &rpcpb.Candidates{
		Candidates: rpcCandidates,
	}, nil
}

// GetDynasty returns all dynasty accounts
func (s *APIService) GetDynasty(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.Dynasty, error) {
	block := s.bm.TailBlock()
	dynastySize := s.bm.Consensus().DynastySize()
	dynasty := make([]common.Address, dynastySize)

	// TODO Fix bug when fewer dynasty
	var err error
	for i := 0; i < dynastySize; i++ {
		dynasty[i], err = block.State().DposState().GetProposer(i)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("failed to get dynasty.")
			return nil, status.Error(codes.Internal, ErrMsgInternalError)
		}
	}

	addresses := make([]string, dynastySize)
	for i, addr := range dynasty {
		addresses[i] = addr.Hex()
	}
	return &rpcpb.Dynasty{
		Addresses: addresses,
	}, nil
}

// GetMedState return mednet state
func (s *APIService) GetMedState(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.MedState,
	error) {
	tailBlock := s.bm.TailBlock()
	lib := s.bm.LIB()

	return &rpcpb.MedState{
		ChainId:   tailBlock.ChainID(),
		Tail:      byteutils.Bytes2Hex(tailBlock.Hash()),
		Height:    tailBlock.Height(),
		Lib:       byteutils.Bytes2Hex(lib.Hash()),
		NetworkId: s.ns.Node().ID().Pretty(),
	}, nil
}

// GetPendingTransactions sends all transactions in the transaction pool
// TODO Need GETALL(?)
func (s *APIService) GetPendingTransactions(ctx context.Context,
	req *rpcpb.NonParamRequest) (*rpcpb.Transactions, error) {
	// txs := s.tm.GetAll()
	// rpcTxs := transactions2rpcTxs(txs, false)
	//
	// return &rpcpb.Transactions{
	// 	Transactions: rpcTxs,
	// }, nil
	return nil, nil
}

// GetTransaction returns transaction
func (s *APIService) GetTransaction(ctx context.Context, req *rpcpb.GetTransactionRequest) (*rpcpb.
	Transaction, error) {
	if !common.IsHash(req.Hash) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTxHash)
	}

	tailBlock := s.bm.TailBlock()
	if tailBlock == nil {
		logging.Console().Error("failed to get tail block.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	txHash, err := byteutils.Hex2Bytes(req.Hash)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"hash": req.Hash,
			"err":  err,
		}).Error("failed to convert hex hash to bytes.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}
	tx, err := tailBlock.State().GetTx(txHash)
	if err != nil && err != trie.ErrNotFound {
		logging.Console().WithFields(logrus.Fields{
			"hash": txHash,
			"err":  err,
		}).Error("failed to get transaction.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	} else if err == trie.ErrNotFound { // tx is not in txsState
		// Get tx from txPool (type *Transaction)
		tx = s.tm.Get(txHash)
		if tx == nil {
			return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
		}
		return CoreTx2rpcTx(tx, false), nil
	}
	// If tx is already included in a block
	return CoreTx2rpcTx(tx, true), nil
}

// GetTransactionReceipt returns transaction receipt
func (s *APIService) GetTransactionReceipt(ctx context.Context, req *rpcpb.GetTransactionRequest) (*rpcpb.
	TransactionReceipt, error) {
	if !common.IsHash(req.Hash) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTxHash)
	}

	tailBlock := s.bm.TailBlock()
	if tailBlock == nil {
		logging.Console().Error("failed to get tail block.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	txHash, err := byteutils.Hex2Bytes(req.Hash)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"hash": req.Hash,
			"err":  err,
		}).Error("failed to convert hex hash to bytes.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}
	tx, err := tailBlock.State().GetTx(txHash)
	if err != nil && err != trie.ErrNotFound {
		logging.Console().WithFields(logrus.Fields{
			"hash": txHash,
			"err":  err,
		}).Error("failed to get transaction.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	} else if err == trie.ErrNotFound { // tx is not in txsState
		return nil, status.Error(codes.NotFound, ErrMsgTransactionNotFound)
	}
	// If tx is already included in a block
	return coreReceipt2rpcReceipt(tx), nil
}

// SendTransaction sends transaction
func (s *APIService) SendTransaction(ctx context.Context, req *rpcpb.SendTransactionRequest) (*rpcpb.
	TransactionHash, error) {
	if !common.IsHash(req.Hash) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTxHash)
	}
	if req.Payload != "" && !byteutils.IsHex(req.Payload) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidPayload)
	}
	if req.To != "" && !common.IsHexAddress(req.To) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidToAddress)
	}
	if req.Sign == "" || !byteutils.IsHex(req.Sign) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidSignature)
	}
	if req.PayerSign != "" && !byteutils.IsHex(req.PayerSign) {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidPayerSignature)
	}

	value, err := util.NewUint128FromString(req.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, ErrMsgInvalidTxValue)
	}
	valueBytes, err := value.ToFixedSizeByteSlice()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"value": req.Value,
			"err":   err,
		}).Error("failed to covert uint128 to fixed size byte slice.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	payloadBuf, err := hex.DecodeString(req.Payload)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"payload": req.Payload,
			"err":     err,
		}).Error("failed to covert payload hex string to byte slice.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	hashBytes, err := byteutils.Hex2Bytes(req.Hash)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"hash": req.Hash,
			"err":  err,
		}).Error("failed to covert hash hex to bytes.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	toBytes, err := byteutils.Hex2Bytes(req.To)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"address": req.To,
			"err":     err,
		}).Error("failed to covert hex address to bytes.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	signBytes, err := byteutils.Hex2Bytes(req.Sign)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"sign": req.Sign,
			"err":  err,
		}).Error("failed to covert hex signature to bytes.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	payerSignBytes, err := byteutils.Hex2Bytes(req.PayerSign)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"payerSign": req.PayerSign,
			"err":       err,
		}).Error("failed to covert hex payer signature to bytes.")
		return nil, status.Error(codes.Internal, ErrMsgInternalError)
	}

	pbTx := &corepb.Transaction{
		Hash:      hashBytes,
		TxType:    req.TxType,
		To:        toBytes,
		Value:     valueBytes,
		Nonce:     req.Nonce,
		ChainId:   req.ChainId,
		Payload:   payloadBuf,
		Sign:      signBytes,
		PayerSign: payerSignBytes,
		Receipt:   nil,
	}

	tx := &core.Transaction{}
	if err := tx.FromProto(pbTx); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.tm.PushAndBroadcast(tx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &rpcpb.TransactionHash{
		Hash: byteutils.Bytes2Hex(tx.Hash()),
	}, nil
}

// Subscribe to listen event
func (s *APIService) Subscribe(req *rpcpb.SubscribeRequest, stream rpcpb.ApiService_SubscribeServer) error {

	eventSub, err := event.NewSubscriber(1024, req.Topics)
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
				Hash:  event.Data,
				Type:  event.Type,
			})
			// TODO : Send timeout
			if err != nil {
				return err
			}
		}
	}
}

// HealthCheck returns success.
func (s *APIService) HealthCheck(ctx context.Context, req *rpcpb.NonParamRequest) (*rpcpb.Health,
	error) {
	return &rpcpb.Health{
		Ok: true,
	}, nil
}
