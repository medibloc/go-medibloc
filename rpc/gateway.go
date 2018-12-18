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
	"net/http"

	"encoding/json"

	"io"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/medibloc/go-medibloc/rpc/pb"
	rpcpb "github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HTTPServer is a rest gateway wrapping the grpc server.
type HTTPServer struct {
	handler  http.Handler
	httpAddr string
	grpcAddr string
	cancel   context.CancelFunc
}

// NewHTTPServer creates HTTPServer.
func NewHTTPServer(httpAddr string, grpcAddr string) (*HTTPServer, error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard,
		&runtime.JSONPb{OrigName: true, EmitDefaults: true}),
		runtime.WithProtoErrorHandler(errorHandler))
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := rpcpb.RegisterApiServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
	if err != nil {
		return nil, err
	}

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(pb.Swagger))
	})

	httpMux.Handle("/", mux)

	return &HTTPServer{
		handler:  cors.Default().Handler(httpMux),
		httpAddr: httpAddr,
		grpcAddr: grpcAddr,
		cancel:   cancel,
	}, nil
}

// Run starts the server.
func (srv *HTTPServer) Run() error {
	defer srv.cancel()
	return http.ListenAndServe(srv.httpAddr, srv.handler)
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
	if s.Code() == codes.Unimplemented {
		s = status.New(codes.NotFound, "Page not found")
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
