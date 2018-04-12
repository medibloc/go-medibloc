package rpc

import (
	"log"
	"net"

	rpcpb "github.com/medibloc/go-medibloc/rpc/proto"
	"google.golang.org/grpc"
)

type GRPCServer interface {
	Start(string) error
	RunGateway() error
	Stop()
}

type Server struct {
	rpcServer *grpc.Server
}

func NewServer() *Server {
	rpc := grpc.NewServer()
	srv := &Server{rpcServer: rpc}
	api := &APIService{server: srv}
	admin := &AdminService{server: srv}
	rpcpb.RegisterApiServiceServer(rpc, api)
	rpcpb.RegisterAdminServiceServer(rpc, admin)
	return srv
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
	}
	go func() {
		if err := s.rpcServer.Serve(lis); err != nil {
			log.Println("Somethins is wrong in Start : %v", err)
		}
	}()
	log.Println("Server is running")
	return nil
}

func (s *Server) RunGateway() error {
	go func() {
		if err := httpServerRun(); err != nil {
			log.Println("Somethins is wrong in RunGateway : %v", err)
		}
	}()
	log.Println("HTTPServer is running")
	return nil
}

func (s *Server) Stop() {
	s.rpcServer.Stop()
}
