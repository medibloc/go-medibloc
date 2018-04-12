package rpc

import (
	"log"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)



func httpServerRun() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err1 := rpcpb.RegisterApiServiceHandlerFromEndpoint(ctx, mux, "localhost:10000", opts)
	err2 := rpcpb.RegisterAdminServiceHandlerFromEndpoint(ctx, mux, "localhost:10000", opts)
	if err1 != nil{
		log.Println("Somethins is wrong in httpServerRun : %v", err1)
	}
	if err2 != nil{
		log.Println("Somethins is wrong in httpServerRun : %v", err2)
	}
	if err := http.ListenAndServe("localhost:10002", mux); err != nil {
		log.Println("Somethins is wrong in httpServerRun : %v", err)
		return err
	}
	return nil
}
