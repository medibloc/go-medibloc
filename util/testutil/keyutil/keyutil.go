package keyutil

import (
	"fmt"
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/stretchr/testify/require"
)

// AddrKeyPair contains address and private key.
type AddrKeyPair struct {
	Addr    common.Address
	PrivKey signature.PrivateKey
}

// NewAddrKeyPairFromPrivKey creates a pair from private key.
func NewAddrKeyPairFromPrivKey(t *testing.T, privKey signature.PrivateKey) *AddrKeyPair {
	addr, err := common.PublicKeyToAddress(privKey.PublicKey())
	require.NoError(t, err)
	return &AddrKeyPair{
		Addr:    addr,
		PrivKey: privKey,
	}
}

// NewAddrKeyPair creates a pair of address and private key.
func NewAddrKeyPair(t *testing.T) *AddrKeyPair {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	require.NoError(t, err)

	return NewAddrKeyPairFromPrivKey(t, privKey)
}

// Address returns address.
func (pair *AddrKeyPair) Address() string {
	return pair.Addr.Hex()
}

// PrivateKey returns private key.
func (pair *AddrKeyPair) PrivateKey() string {
	d, _ := pair.PrivKey.Encoded()
	return byteutils.Bytes2Hex(d)
}

// String describes AddrKeyPair in string format.
func (pair *AddrKeyPair) String() string {
	if pair == nil {
		return ""
	}
	return fmt.Sprintf("Addr:%v, PrivKey:%v\n", pair.Address(), pair.PrivateKey())
}

// AddrKeyPairs is a slice of AddrKeyPair structure.
type AddrKeyPairs []*AddrKeyPair

// FindPrivKey finds private key of given address.
func (pairs AddrKeyPairs) FindPrivKey(addr common.Address) signature.PrivateKey {
	for _, dynasty := range pairs {
		if dynasty.Addr.Equals(addr) {
			return dynasty.PrivKey
		}
	}
	return nil
}

// FindPair finds AddrKeyPair of given address.
func (pairs AddrKeyPairs) FindPair(addr common.Address) *AddrKeyPair {
	for _, dynasty := range pairs {
		if dynasty.Addr.Equals(addr) {
			return dynasty
		}
	}
	return nil
}
