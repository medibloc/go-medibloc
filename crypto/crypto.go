package crypto

import (
  "errors"
  "github.com/medibloc/go-medibloc/crypto/secp256k1"
  "github.com/medibloc/go-medibloc/keystore"
)

var (
  // ErrAlgorithmInvalid invalid Algorithm for sign.
  ErrAlgorithmInvalid = errors.New("invalid Algorithm")
)

func GenerateKey(alg keystore.Algorithm) (keystore.PrivateKey, error) {
  switch alg {
  case keystore.SECP256K1:
    return secp256k1.GeneratePrivateKey(), nil
  default:
    return nil, ErrAlgorithmInvalid
  }
}

func NewSignature(alg keystore.Algorithm) (keystore.Signature, error) {
  switch alg {
  case keystore.SECP256K1:
    return new(secp256k1.Signature), nil
  default:
    return nil, ErrAlgorithmInvalid
  }
}

func CheckAlgorithm(alg keystore.Algorithm) error {
  switch alg {
  case keystore.SECP256K1:
    return nil
  default:
    return ErrAlgorithmInvalid
  }
}
