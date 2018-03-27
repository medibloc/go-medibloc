package secp256k1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/math"
	"golang.org/x/crypto/sha3"
)

var (
	secp256k1_N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1_halfN = new(big.Int).Div(secp256k1_N, big.NewInt(2))
)

// NewECDSAPrivateKey generate a ecdsa private key
func NewECDSAPrivateKey() *ecdsa.PrivateKey {
	var priv *ecdsa.PrivateKey
	for {
		priv, _ = ecdsa.GenerateKey(S256(), rand.Reader)
		if SeckeyVerify(FromECDSAPrivateKey(priv)) {
			break
		}
	}
	return priv
}

// FromECDSA exports a private key into a binary dump.
func FromECDSAPrivateKey(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return ToECDSAPrivateKey(b)
}

func ToECDSAPrivateKey(d []byte) (*ecdsa.PrivateKey, error) {
	return toECDSAPrivateKey(d, true)
}

func ToECDSAPrivateKeyUnsafe(d []byte) *ecdsa.PrivateKey {
	priv, _ := toECDSAPrivateKey(d, false)
	return priv
}

func toECDSAPrivateKey(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = S256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1_N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

func FromECDSAPublicKey(pub *ecdsa.PublicKey) ([]byte, error) {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil, errors.New("invalid public key input")
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y), nil
}

func ToECDSAPublicKey(pub []byte) (*ecdsa.PublicKey, error) {
	if len(pub) == 0 {
		return nil, errors.New("invalid public key")
	}
	x, y := elliptic.Unmarshal(S256(), pub)
	return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}, nil
}

func PubkeyToAddress(p ecdsa.PublicKey) (common.Address, error) {
	pubBytes, err := FromECDSAPublicKey(&p)
	if err != nil {
		return common.Address{}, err
	}
	hash := sha3.Sum256(pubBytes[1:])
	return common.BytesToAddress(hash[12:]), nil
}

// zeroKey zeroes the private key
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}
