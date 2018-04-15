package rpc

import (
	"log"
	"net"

	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"google.golang.org/grpc"
)

// GRPCServer is GRPCServer's interface.
type GRPCServer interface {
	Start(string) error
	RunGateway() error
	Stop()
}

// Server is rpc server.
type Server struct {
	rpcServer *grpc.Server
}

// NewServer returns NewServer.
func NewServer() *Server {
	rpc := grpc.NewServer()
	srv := &Server{rpcServer: rpc}
	api := &APIService{server: srv}
	admin := &AdminService{server: srv}
	rpcpb.RegisterApiServiceServer(rpc, api)
	rpcpb.RegisterAdminServiceServer(rpc, admin)
	return srv
}

// Start starts rpc server.
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
	}
	go func() {
		if err := s.rpcServer.Serve(lis); err != nil {
			log.Printf("Somethins is wrong in Start : %v", err)
		}
	}()
	log.Println("Server is running")
	return nil
}

// RunGateway runs rest gateway server.
func (s *Server) RunGateway() error {
	go func() {
		if err := httpServerRun(); err != nil {
			log.Printf("Somethins is wrong in RunGateway : %v", err)
		}
	}()
	log.Println("HTTPServer is running")
	return nil
}

// Stop stops server.
func (s *Server) Stop() {
	s.rpcServer.Stop()
}
