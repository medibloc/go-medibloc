package secp256k1

import (
	"crypto/ecdsa"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

// PrivateKey ecdsa privatekey
type PrivateKey struct {
	privateKey *ecdsa.PrivateKey
}

// NewPrivateKey new a private key with ecdsa.PrivateKey
func NewPrivateKey(key *ecdsa.PrivateKey) *PrivateKey {
	priv := new(PrivateKey)
	priv.privateKey = key
	return priv
}

// GeneratePrivateKey generate a new private key
func GeneratePrivateKey() *PrivateKey {
	priv := new(PrivateKey)
	ecdsa := NewECDSAPrivateKey()
	priv.privateKey = ecdsa
	return priv
}

//NewPrivateKeyFromHex gets new private key from hex string.
func NewPrivateKeyFromHex(b string) (*PrivateKey, error) {
	ecdsaKey, err := HexToECDSA(b)
	if err != nil {
		return nil, err
	}
	privKey := NewPrivateKey(ecdsaKey)
	return privKey, nil
}

// Algorithm returns algorithm name.
func (k *PrivateKey) Algorithm() algorithm.Algorithm {
	return algorithm.SECP256K1
}

// Encoded encodes to bytes.
func (k *PrivateKey) Encoded() ([]byte, error) {
	return FromECDSAPrivateKey(k.privateKey), nil
}

// Decode decode data to key.
func (k *PrivateKey) Decode(b []byte) error {
	priv, err := ToECDSAPrivateKey(b)
	if err != nil {
		return err
	}
	k.privateKey = priv
	return nil
}

// Clear clear key content.
func (k *PrivateKey) Clear() {
	zeroKey(k.privateKey)
}

// PublicKey returns public key.
func (k *PrivateKey) PublicKey() signature.PublicKey {
	return NewPublicKey(k.privateKey.PublicKey)
}

// Sign sing bytes with private key.
func (k *PrivateKey) Sign(msg []byte) ([]byte, error) {
	return Sign(msg, FromECDSAPrivateKey(k.privateKey))
}
