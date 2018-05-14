package rpc

import (
	"net/http"

	"encoding/json"

	"io"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/medibloc/go-medibloc/rpc/pb"
	pb "github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(pb.Swagger))
	})

	httpMux.Handle("/", mux)

	handler := cors.Default().Handler(httpMux)
	return http.ListenAndServe(addr, handler)
}

type errorBody struct {
	Error string `json:"error,omitempty"`
}

func errorHandler(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error) {
	const fallback = "failed to marshal error message"

	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}
	body := &errorBody{
		Error: s.Message(),
	}

	w.Header().Set("Content-type", marshaler.ContentType())
	if s.Code() == codes.Unknown {
		w.WriteHeader(runtime.HTTPStatusFromCode(codes.OutOfRange))
	} else {
		w.WriteHeader(runtime.HTTPStatusFromCode(s.Code()))
	}

	jErr := json.NewEncoder(w).Encode(errorBody{
		Error: body.Error,
	})

	if jErr != nil {
		jsonFallback, tmpErr := json.Marshal(errorBody{Error: fallback})
		if tmpErr != nil {
			logging.WithFields(logrus.Fields{
				"error":        tmpErr,
				"jsonFallback": jsonFallback,
			}).Debug("Failed to marshal fallback msg")
		}
		_, tmpErr = w.Write(jsonFallback)
		if tmpErr != nil {
			logging.WithFields(logrus.Fields{
				"error": tmpErr,
			}).Debug("Failed to write fallback msg")
		}
	}
}
