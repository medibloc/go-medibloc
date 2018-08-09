package rpc

import (
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-nebulas/util"
)

func coreAccount2rpcAccount(account *core.Account) (*rpcpb.GetAccountResponse, error) {
	var txsFrom []string
	var txsTo []string
	for _, hash := range acc.TxsFrom() {
		txsFrom = append(txsFrom, byteutils.Bytes2Hex(hash))
	}
	for _, hash := range acc.TxsFrom() {
		txsTo = append(txsFrom, byteutils.Bytes2Hex(hash))
	}

	return &rpcpb.GetAccountResponse{
		Address:       byteutils.Bytes2Hex(account.Address()),
		Balance:       account.Balance().String(),
		Nonce:         account.Nonce(),
		Vesting:       account.Vesting().String(),
		Voted:         byteutils.Bytes2Hex(acc.Voted()),
		Records:       nil, // TODO @ggomma
		CertsIssued:   nil, // TODO @ggomma
		CertsReceived: nil, // TODO @ggomma
		TxsFrom:       txsFrom,
		TxsTo:         txsTo,
	}, nil
}

func coreBlock2rpcBlock(block *corepb.Block) (*rpcpb.GetBlockResponse, error) {
	var rpcTxs []*rpcpb.GetTransactionResponse
	for _, tx := range block.GetTransactions() {
		rpcTx, err := coreTx2rpcTx(tx, true)
		if err != nil {
			return nil, err
		}
		rpcTxs = append(rpcTxs, rpcTx)
	}

	return &rpcpb.BlockResponse{
		Height:            block.Height,
		Hash:              byteutils.Bytes2Hex(block.Header.Hash),
		ParentHash:        byteutils.Bytes2Hex(block.Header.ParentHash),
		Coinbase:          byteutils.Bytes2Hex(block.Header.Coinbase),
		Reward:            nil, // todo
		Supply:            nil, //todo
		Timestamp:         block.Header.Timestamp,
		ChainId:           block.Header.ChainId,
		Alg:               block.Header.Alg,
		Sign:              byteutils.Bytes2Hex(block.Header.Sign),
		AccsRoot:          byteutils.Bytes2Hex(block.Header.AccsRoot),
		TxsRoot:           byteutils.Bytes2Hex(block.Header.TxsRoot),
		UsageRoot:         byteutils.Bytes2Hex(block.Header.UsageRoot),
		RecordsRoot:       byteutils.Bytes2Hex(block.Header.RecordsRoot),
		CertificationRoot: byteutils.Bytes2Hex(block.Header.DposRoot),
		DposRoot:          nil, //todo
		Transactions:      rpcPbTxs,
	}, nil
}

func dposCandidate2rpcCandidate(candidate *dpospb.Candidate) (*rpcpb.Candidate, error) {
	collateral, err := util.NewUint128FromFixedSizeByteSlice(candidate.Collateral)
	if err != nil {
		return nil, err
	}

	votePower, err := util.NewUint128FromFixedSizeByteSlice(candidate.VotePower)
	if err != nil {
		return nil, err
	}

	return &rpcpb.Candidate{
		Address:   byteutils.Bytes2Hex(candidate.Address),
		Collatral: collatral.String(),
		VotePower: votePower.String(),
	}, nil
}

func coreTx2rpcTx(tx *corepb.Transaction, executed bool) (*rpcpb.GetTransactionResponse, error) {
	value, err := util.NewUint128FromFixedSizeByteSlice(tx.Value)
	if err != nil {
		return nil, err
	}

	return &rpcpb.GetTransactionResponse{
		Hash:      byteutils.Bytes2Hex(tx.Hash),
		From:      byteutils.Bytes2Hex(tx.From),
		To:        byteutils.Bytes2Hex(tx.To),
		Value:     value.String(),
		Timestamp: tx.Timestamp,
		Data: &rpcpb.TransactionData{
			Type:    tx.Data.Type,
			Payload: string(tx.Data.Payload),
		},
		Nonce:     tx.Nonce,
		ChainId:   tx.ChainId,
		Alg:       tx.Alg,
		Sign:      byteutils.Bytes2Hex(tx.Sign),
		PayerSign: byteutils.Bytes2Hex(tx.PayerSign),
		Executed:  executed,
	}, nil
}
