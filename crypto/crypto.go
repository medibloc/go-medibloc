package crypto

import (
	"errors"

	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
)

var (
	// ErrAlgorithmInvalid invalid Algorithm for sign.
	ErrAlgorithmInvalid = errors.New("invalid Algorithm")
)

func GenerateKey(alg algorithm.Algorithm) (signature.PrivateKey, error) {
	switch alg {
	case algorithm.SECP256K1:
		return secp256k1.GeneratePrivateKey(), nil
	default:
		return nil, ErrAlgorithmInvalid
	}
}

func NewSignature(alg algorithm.Algorithm) (signature.Signature, error) {
	switch alg {
	case algorithm.SECP256K1:
		return new(secp256k1.Signature), nil
	default:
		return nil, ErrAlgorithmInvalid
	}
}

func CheckAlgorithm(alg algorithm.Algorithm) error {
	switch alg {
	case algorithm.SECP256K1:
		return nil
	default:
		return ErrAlgorithmInvalid
	}
}
