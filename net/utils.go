package net

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Errors
var (
	ErrListenPortIsNotAvailable = errors.New("listen port is not available")
	ErrConfigLackNetWork        = errors.New("config.conf should has network")
)

// ParseFromIPFSAddr return pid and address parsed from ipfs address
func ParseFromIPFSAddr(ipfsAddr ma.Multiaddr) (peer.ID, ma.Multiaddr, error) {
	addr, err := ma.NewMultiaddr(strings.Split(ipfsAddr.String(), "/ipfs/")[0])
	if err != nil {
		return "", nil, err
	}

	// TODO: we should register med multicodecs.
	b58, err := ipfsAddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		return "", nil, err
	}

	id, err := peer.IDB58Decode(b58)
	if err != nil {
		return "", nil, err
	}

	return id, addr, nil
}

func verifyListenAddress(listen []string) error {
	for _, v := range listen {
		_, err := net.ResolveTCPAddr("tcp", v)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkPathConfig(path string) bool {
	if path == "" {
		return true
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

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
