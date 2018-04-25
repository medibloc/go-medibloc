package rpc

import (
	"net"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet/pb"
	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"github.com/medibloc/go-medibloc/util/logging"
	"google.golang.org/grpc"
)

// Server is rpc server.
type Server struct {
	addrGrpc  string
	addrHTTP  string
	rpcServer *grpc.Server
}

// New returns NewServer.
func New(cfg *medletpb.Config) *Server {
	rpc := grpc.NewServer()
	return &Server{
		rpcServer: rpc,
		addrGrpc:  cfg.Rpc.RpcListen[0],
		addrHTTP:  cfg.Rpc.HttpListen[0],
	}
}

//Setup sets up server.
func (s *Server) Setup(bm *core.BlockManager, tm *core.TransactionManager) {
	api := newAPIService(bm, tm)
	admin := newAdminService(bm, tm)
	rpcpb.RegisterApiServiceServer(s.rpcServer, api)
	rpcpb.RegisterAdminServiceServer(s.rpcServer, admin)
}

// Start starts rpc server.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addrGrpc)
	if err != nil {
		return err
	}
	go func() {
		if err := s.rpcServer.Serve(lis); err != nil {
			logging.Console().Error(err)
		}
	}()
	logging.Console().Info("GRPC Server is running...")

	s.RunGateway()
	return nil
}

// RunGateway runs rest gateway server.
func (s *Server) RunGateway() error {
	go func() {
		if err := httpServerRun(s.addrHTTP, s.addrGrpc); err != nil {
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
