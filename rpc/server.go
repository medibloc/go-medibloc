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
	goNet "net"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/event"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/net"
	rpcpb "github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
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

// Setup sets up server.
func (s *Server) Setup(bm *core.BlockManager, tm *core.TransactionManager, ee *event.Emitter, ns net.Service) {
	api := newAPIService(bm, tm, ee, ns)
	rpcpb.RegisterApiServiceServer(s.rpcServer, api)
}

// Start starts rpc server.
func (s *Server) Start() error {
	lis, err := goNet.Listen("tcp", s.addrGrpc)
	if err != nil {
		return err
	}
	go func() {
		if err := s.rpcServer.Serve(lis); err != nil {
			logging.Console().Error(err)
		}
	}()
	logging.Console().Info("GRPC Server is running...")

	err = s.RunGateway()
	if err != nil {
		return err
	}
	return nil
}

// RunGateway runs rest gateway server.
func (s *Server) RunGateway() error {
	httpServer, err := NewHTTPServer(s.addrHTTP, s.addrGrpc)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to create GRPC HTTP Gateway.")
	}
	go func() {
		err = httpServer.Run()
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Error("Failed to run GRPC HTTP Gateway.")
		}
	}()
	logging.Console().Info("GRPC HTTP Gateway is running...")
	return nil
}

// Stop stops server.
func (s *Server) Stop() {
	s.rpcServer.Stop()
}
