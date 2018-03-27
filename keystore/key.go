package keystore

import (
	"github.com/medibloc/go-medibloc/common"
)

const (
	// version compatible with ethereum, the version start with 3
	version = 3
)

// Algorithm type alias
type Algorithm uint8

const (
	// SECP256K1 a type of signer
	SECP256K1 Algorithm = 1
)

// Key interface
type Key interface {

	// Algorithm returns the standard algorithm for this key. For
	// example, "ECDSA" would indicate that this key is a ECDSA key.
	Algorithm() Algorithm

	Address() (common.Address, error)

	// Encoded returns the key in its primary encoding format, or null
	// if this key does not support encoding.
	Encoded() ([]byte, error)

	// Decode decode data to key
	Decode(data []byte) error

	// Clear clear key content
	Clear()
}

// PrivateKey privatekey interface
type PrivateKey interface {

	// Algorithm returns the standard algorithm for this key. For
	// example, "ECDSA" would indicate that this key is a ECDSA key.
	Algorithm() Algorithm

	Address() (common.Address, error)

	// Encoded returns the key in its primary encoding format, or null
	// if this key does not support encoding.
	Encoded() ([]byte, error)

	// Decode decode data to key
	Decode(data []byte) error

	// Clear clear key content
	Clear()

	// PublicKey returns publickey
	PublicKey() PublicKey
}

// PublicKey publickey interface
type PublicKey interface {

	// Algorithm returns the standard algorithm for this key. For
	// example, "ECDSA" would indicate that this key is a ECDSA key.
	Algorithm() Algorithm

	Address() (common.Address, error)

	// Encoded returns the key in its primary encoding format, or null
	// if this key does not support encoding.
	Encoded() ([]byte, error)

	// Decode decode data to key
	Decode(data []byte) error

	// Clear clear key content
	Clear()
}
