package main

import (
	"flag"
	"log"

	"github.com/medibloc/go-medibloc/rpc"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"golang.org/x/net/context"
)

var (
	ttype   = flag.String("ttype", "", "Type admin or api")
	address = flag.String("addr", "", "Address")
	height  = flag.Int("height", 1, "Height")
	from    = flag.String("from", "", "From")
	to      = flag.String("to", "", "To")
	value   = flag.String("amount", "", "Amount")
	nonce   = flag.Int("nonce", 0, "Nonce")
)

func main() {
	flag.Parse()
	addr := "localhost:10000"
	conn := rpc.Dial(addr)
	defer conn.Close()

	apiClient := rpcpb.NewApiServiceClient(conn)
	adminClient := rpcpb.NewAdminServiceClient(conn)
	switch *ttype {
	case "admin":
		res, err := adminClient.SendTransaction(context.Background(), &rpcpb.TransactionRequest{
			From:  *from,
			To:    *to,
			Value: *value,
			Nonce: uint64(*nonce),
		})
		if err != nil {
			log.Printf("Somethins is wrong in client main : %v", err)
		}
		log.Println(res)
	case "api":
		res, err := apiClient.GetAccountState(context.Background(), &rpcpb.GetAccountStateRequest{
			Address: *address,
			Height:  uint64(*height),
		})
		if err != nil {
			log.Printf("Somethins is wrong in client main : %v", err)
		}
		log.Println(res)
	}
}
