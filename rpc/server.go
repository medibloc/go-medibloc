package rpc

import (
	"net"

	"github.com/medibloc/go-medibloc/core"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"github.com/medibloc/go-medibloc/util/logging"
	"google.golang.org/grpc"
)

// Bridge interface for getters
type Bridge interface {
	// BlockManager return core.BlockManager
	BlockManager() *core.BlockManager

	// TransactionManager return core.TransactionManager
	TransactionManager() *core.TransactionManager
}

// GRPCServer is GRPCServer's interface.
type GRPCServer interface {
	Start() error
	RunGateway(string) error
	Stop()
}

// Server is rpc server.
type Server struct {
	addrGrpc  string
	rpcServer *grpc.Server
}

// NewServer returns NewServer.
func NewServer(bridge Bridge, addrGrpc string) GRPCServer {
	rpc := grpc.NewServer()
	srv := &Server{rpcServer: rpc, addrGrpc: addrGrpc}
	api := &APIService{bridge: bridge, server: srv}
	admin := &AdminService{bridge: bridge, server: srv}
	rpcpb.RegisterApiServiceServer(rpc, api)
	rpcpb.RegisterAdminServiceServer(rpc, admin)
	return srv
}

// Start starts rpc server.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addrGrpc)
	if err != nil {
	}
	go func() {
		if err := s.rpcServer.Serve(lis); err != nil {
			logging.Console().Error(err)
		}
	}()
	logging.Console().Info("GRPC Server is running...")
	return nil
}

// RunGateway runs rest gateway server.
func (s *Server) RunGateway(addrHTTP string) error {
	go func() {
		if err := httpServerRun(addrHTTP, s.addrGrpc); err != nil {
			logging.Console().Error(err)
		}
	}()
	logging.Console().Info("GRPC HTTP Gateway is running...")
	return nil
}

// Stop stops server.
func (s *Server) Stop() {
	s.rpcServer.Stop()
}
