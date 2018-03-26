package crypto

import (
  "github.com/medibloc/go-medibloc/common"
  "golang.org/x/crypto/sha3"
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
