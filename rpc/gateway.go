package rpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func httpServerRun(addr string, addrGrpc string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard,
		&runtime.JSONPb{OrigName: true, EmitDefaults: true}),
		runtime.WithProtoErrorHandler(errorHandler))
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

type errorBody struct {
	Err string `json:"error,omitempty"`
}

func errorHandler(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error) {

	w.Header().Set("Content-type", marshaler.ContentType())
	if grpc.Code(err) == codes.Unknown {
		w.WriteHeader(runtime.HTTPStatusFromCode(codes.OutOfRange))
	} else {
		w.WriteHeader(runtime.HTTPStatusFromCode(grpc.Code(err)))
	}
}
