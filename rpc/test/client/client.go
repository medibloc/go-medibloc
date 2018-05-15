package main

import (
	"flag"
	"log"

	"github.com/medibloc/go-medibloc/rpc"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"golang.org/x/net/context"
)

var (
	ttype   = flag.String("ttype", "", "Type admin or api")
	address = flag.String("addr", "", "Address")
	height  = flag.String("height", "1", "Height")
)

func main() {
	flag.Parse()
	addr := "localhost:9920"
	conn := rpc.Dial(addr)
	defer conn.Close()

	apiClient := rpcpb.NewApiServiceClient(conn)
	switch *ttype {
	case "accountstate":
		res, err := apiClient.GetAccountState(context.Background(), &rpcpb.GetAccountStateRequest{
			Address: *address,
			Height:  *height,
		})
		if err != nil {
			log.Printf("Somethins is wrong in client main : %v", err)
		}
		log.Println(res)
	case "medstate":
		res, err := apiClient.GetMedState(context.Background(), &rpcpb.NonParamsRequest{})
		if err != nil {
			log.Println(err)
		} else {
			log.Println(res)
		}
	}
}
