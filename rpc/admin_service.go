package rpc

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"

	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"golang.org/x/net/context"
)

//
// FAKE DATA BEGIN
//

//
// FAKE DATA ENDS
//

type AdminService struct {
	server GRPCServer
}

func (s *AdminService) SendTransaction(ctx context.Context, req *rpcpb.TransactionRequest) (*rpcpb.TransactionResponse, error) {
	data := strings.Join([]string{req.From, req.To, req.Value, string(req.Nonce)}, "")
	hasher := sha256.New()
	hasher.Write([]byte(data))

	txHash := hex.EncodeToString(hasher.Sum(nil))
	log.Println(txHash)

	return &rpcpb.TransactionResponse{
		Txhash: txHash,
	}, nil
}
