package crypto

import (
  "crypto/ecdsa"
  "crypto/elliptic"
  "crypto/rand"
  "encoding/hex"
  "errors"
  "fmt"
  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/common/math"
  "golang.org/x/crypto/sha3"
  "math/big"
)

var (
  secp256k1_N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
  secp256k1_halfN = new(big.Int).Div(secp256k1_N, big.NewInt(2))
)

func Sha3256(data ...[]byte) []byte {
  d := sha3.New256()
  for _, b := range data {
    d.Write(b)
  }
  hash := d.Sum(nil)
  return hash[:]
}

func Sha3256Hash(data ...[]byte) (h common.Hash) {
  d := sha3.New256()
  for _, b := range data {
    d.Write(b)
  }
  hash := d.Sum(nil)
  return common.BytesToHash(hash[:])
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
  return ecdsa.GenerateKey(S256(), rand.Reader)
}

// FromECDSA exports a private key into a binary dump.
func FromECDSA(priv *ecdsa.PrivateKey) []byte {
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
  return ToECDSA(b)
}

func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
  return toECDSA(d, true)
}

func ToECDSAUnsafe(d []byte) *ecdsa.PrivateKey {
  priv, _ := toECDSA(d, false)
  return priv
}

func toECDSA(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
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

  // The priv.D must not b zero or negative
  if priv.D.Sign() <= 0 {
    return nil, fmt.Errorf("invalid private key, zero or negative")
  }

  priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
  if priv.PublicKey.X == nil {
    return nil, errors.New("invalid private key")
  }
  return priv, nil
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
  if pub == nil || pub.X == nil || pub.Y == nil {
    return nil
  }
  return elliptic.Marshal(S256(), pub.X, pub.Y)
}

func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
  if len(pub) == 0 {
    return nil
  }
  x, y := elliptic.Unmarshal(S256(), pub)
  return &ecdsa.PublicKey{Curve: S256(), X: x, Y: y}
}

func PubkeyToAddress(p ecdsa.PublicKey) common.Address {
  pubBytes := FromECDSAPub(&p)
  return common.BytesToAddress(Sha3256(pubBytes[1:])[12:])
}

func zeroBytes(bytes []byte) {
  for i := range bytes {
    bytes[i] = 0
  }
}
