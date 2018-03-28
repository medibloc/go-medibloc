package signature

import (
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

const (
	// version compatible with ethereum, the version start with 3
	version = 3
)

// Key interface
type Key interface {

	// Algorithm returns the standard algorithm for this key. For
	// example, "ECDSA" would indicate that this key is a ECDSA key.
	Algorithm() algorithm.Algorithm

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
	Algorithm() algorithm.Algorithm

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
	Algorithm() algorithm.Algorithm

	// Encoded returns the key in its primary encoding format, or null
	// if this key does not support encoding.
	Encoded() ([]byte, error)

	// Decode decode data to key
	Decode(data []byte) error

	// Clear clear key content
	Clear()
}
