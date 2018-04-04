package net

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"fmt"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/medibloc/go-medibloc/util/logging"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
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

func convertListenAddrToMultiAddr(listen []string) ([]ma.Multiaddr, error) {

	multiaddrs := make([]ma.Multiaddr, len(listen))
	for idx, v := range listen {
		tcpAddr, err := net.ResolveTCPAddr("tcp", v)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":            err,
				"listen address": v,
			}).Error("Invalid listen address.")
			return nil, err
		}

		addr, err := ma.NewMultiaddr(
			fmt.Sprintf(
				"/ip4/%s/tcp/%d",
				tcpAddr.IP,
				tcpAddr.Port,
			),
		)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err":            err,
				"listen address": v,
			}).Error("Invalid listen address.")
			return nil, err
		}

		multiaddrs[idx] = addr
	}

	return multiaddrs, nil
}
