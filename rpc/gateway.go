package rpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func httpServerRun(addr string, addrGrpc string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := rpcpb.RegisterApiServiceHandlerFromEndpoint(ctx, mux, addrGrpc, opts)
	if err != nil {
		return err
	}
	err = rpcpb.RegisterAdminServiceHandlerFromEndpoint(ctx, mux, addrGrpc, opts)
	if err != nil {
		return err
	}
	return http.ListenAndServe(addr, mux)
}
