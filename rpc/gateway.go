package rpc

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/rs/cors"
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

	handler := cors.Default().Handler(mux)
	return http.ListenAndServe(addr, handler)
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
