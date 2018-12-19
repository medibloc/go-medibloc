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

package net

import (
	"errors"
	"net"
	"time"

	peerstore "github.com/libp2p/go-libp2p-peerstore"
	netpb "github.com/medibloc/go-medibloc/net/pb"
)

// Errors
var (
	ErrListenPortIsNotAvailable = errors.New("listen port is not available")
	ErrConfigLackNetWork        = errors.New("config.conf should have network")
)

func checkPortAvailable(listen []string) error {
	for _, v := range listen {
		conn, err := net.DialTimeout("tcp", v, time.Second*1)
		if err == nil {
			conn.Close()
			return ErrListenPortIsNotAvailable
		}
	}
	return nil
}

// PeersToProto convert peers to peer list for config
func PeersToProto(peers ...peerstore.PeerInfo) []*netpb.PeerInfo {
	pbPeers := make([]*netpb.PeerInfo, len(peers))
	for i, p := range peers {
		pi := &netpb.PeerInfo{
			Id:    p.ID.Pretty(),
			Addrs: make([]string, len(p.Addrs)),
		}
		for j, addr := range p.Addrs {
			pi.Addrs[j] = addr.String()
		}
		pbPeers[i] = pi
	}
	return pbPeers
}

// PeerInfoToProto convert peerinfo to peerinfo for config file
func PeerInfoToProto(p peerstore.PeerInfo) *netpb.PeerInfo {
	pb := new(netpb.PeerInfo)
	pb.Id = p.ID.Pretty()

	addrs := p.Addrs
	pb.Addrs = make([]string, len(addrs))
	for i, addr := range addrs {
		pb.Addrs[i] = addr.String()
	}
	return pb
}
