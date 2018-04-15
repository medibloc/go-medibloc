package secp256k1

import (
	"crypto/ecdsa"

	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
)

type PublicKey struct {
	publicKey ecdsa.PublicKey
}

func NewPublicKey(pub ecdsa.PublicKey) *PublicKey {
	ecdsaPub := new(PublicKey)
	ecdsaPub.publicKey = pub
	return ecdsaPub
}

func (k *PublicKey) Algorithm() algorithm.Algorithm {
	return algorithm.SECP256K1
}

func (k *PublicKey) Encoded() ([]byte, error) {
	return FromECDSAPublicKey(&k.publicKey)
}

func (k *PublicKey) Decode(b []byte) error {
	pub, err := ToECDSAPublicKey(b)
	if err != nil {
		return err
	}
	k.publicKey = *pub
	return nil
}

func (k *PublicKey) Clear() {
	k.publicKey = ecdsa.PublicKey{}
}

func (k *PublicKey) Verify(msg []byte, sig []byte) (bool, error) {
	pub, err := k.Encoded()
	if err != nil {
		return false, err
	}
	return VerifySignature(pub, msg, sig), nil
}

func (k *PublicKey) Compressed() ([]byte, error) {
	return CompressPubkey(k.publicKey.X, k.publicKey.Y)
}

func (k *PublicKey) Decompress(data []byte) error {
	x, y, err := DecompressPubkey(data)
	if err != nil {
		return err
	}

	k.publicKey = ecdsa.PublicKey{
		Curve: S256(), X: x, Y: y,
	}

	return nil
}
