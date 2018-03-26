package secp256k1

import (
  "crypto/ecdsa"
  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/keystore"
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

func (k *PrivateKey) Algorithm() keystore.Algorithm {
  return keystore.SECP256K1
}

func (k *PrivateKey) Address() (common.Address, error) {
  return PubkeyToAddress(k.privateKey.PublicKey)
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

func (k *PrivateKey) PublicKey() keystore.PublicKey {
  return NewPublicKey(k.privateKey.PublicKey)
}

func (k *PrivateKey) Sign(msg []byte) ([]byte, error) {
  return Sign(msg, FromECDSAPrivateKey(k.privateKey))
}
