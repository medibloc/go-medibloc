package secp256k1

import (
	"crypto/ecdsa"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

type PrivateKey struct {
	privateKey *ecdsa.PrivateKey
}

func NewPrivateKey(key *ecdsa.PrivateKey) *PrivateKey {
	priv := new(PrivateKey)
	priv.privateKey = key
	return priv
}

func GeneratePrivateKey() *PrivateKey {
	priv := new(PrivateKey)
	ecdsa := NewECDSAPrivateKey()
	priv.privateKey = ecdsa
	return priv
}

func NewPrivateKeyFromHex(b string) (*PrivateKey, error) {
	ecdsaKey, err := HexToECDSA(b)
	if err != nil {
		return nil, err
	}
	privKey := NewPrivateKey(ecdsaKey)
	return privKey, nil
}

func (k *PrivateKey) Algorithm() algorithm.Algorithm {
	return algorithm.SECP256K1
}

func (k *PrivateKey) Encoded() ([]byte, error) {
	return FromECDSAPrivateKey(k.privateKey), nil
}

func (k *PrivateKey) Decode(b []byte) error {
	priv, err := ToECDSAPrivateKey(b)
	if err != nil {
		return err
	}
	k.privateKey = priv
	return nil
}

func (k *PrivateKey) Clear() {
	zeroKey(k.privateKey)
}

func (k *PrivateKey) PublicKey() signature.PublicKey {
	return NewPublicKey(k.privateKey.PublicKey)
}

func (k *PrivateKey) Sign(msg []byte) ([]byte, error) {
	return Sign(msg, FromECDSAPrivateKey(k.privateKey))
}
