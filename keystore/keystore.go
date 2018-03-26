// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
  "bytes"
  "crypto/aes"
  crand "crypto/rand"
  "encoding/hex"
  "encoding/json"
  "errors"
  "fmt"
  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/common/math"
  "github.com/medibloc/go-medibloc/crypto"
  "github.com/medibloc/go-medibloc/crypto/rand"
  "github.com/pborman/uuid"
  "golang.org/x/crypto/scrypt"
)

const (
  keyHeaderKDF = "scrypt"

  // StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
  // memory and taking approximately 1s CPU time on a modern processor.
  StandardScryptN = 1 << 18

  // StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
  // memory and taking approximately 1s CPU time on a modern processor.
  StandardScryptP = 1

  // LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
  // memory and taking approximately 100ms CPU time on a modern processor.
  LightScryptN = 1 << 12

  // LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
  // memory and taking approximately 100ms CPU time on a modern processor.
  LightScryptP = 6

  scryptR     = 8
  scryptDKLen = 32
)

var (
  ErrNoMatch = errors.New("no key for given address")
  ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

type KeyStore struct {
  keys    map[common.Address][]byte
  scryptN int
  scryptP int
}

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore() *KeyStore {
  ks := &KeyStore{
    scryptN: StandardScryptN,
    scryptP: StandardScryptP,
  }
  ks.init()
  return ks
}

func (ks *KeyStore) init() {
  ks.keys = make(map[common.Address][]byte)
}

// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (ks *KeyStore) NewAccount(auth string) (common.Address, error) {
  key, err := newKey(crand.Reader)
  if err != nil {
    return common.Address{}, err
  }
  keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
  if err != nil {
    return common.Address{}, err
  }
  ks.keys[key.Address] = keyjson
  return key.Address, nil
}

func (ks *KeyStore) Delete(a common.Address) error {
  if !ks.HasAddress(a) {
    return ErrNoMatch
  }
  delete(ks.keys, a)
  return nil
}

// HasAddress reports whether a key with the given address is present.
func (ks *KeyStore) HasAddress(addr common.Address) bool {
  return len(ks.keys[addr]) > 0
}

// Accounts returns all key files present in the directory.
func (ks *KeyStore) Accounts() []common.Address {
  addresses := []common.Address{}
  for addr := range ks.keys {
    addresses = append(addresses, addr)
  }
  return addresses
}

func EncryptKey(key *Key, auth string, scryptN, scryptP int) ([]byte, error) {
  authArray := []byte(auth)
  salt := rand.GetEntropyCSPRNG(32)
  derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
  if err != nil {
    return nil, err
  }
  encryptKey := derivedKey[:16]
  keyBytes := math.PaddedBigBytes(key.PrivateKey.D, 32)

  iv := rand.GetEntropyCSPRNG(aes.BlockSize)
  cipherText, err := crypto.AESCTRXOR(encryptKey, keyBytes, iv)
  if err != nil {
    return nil, err
  }
  mac := crypto.Sha3256(derivedKey[16:32], cipherText)

  scryptParamsJSON := make(map[string]interface{}, 5)
  scryptParamsJSON["n"] = scryptN
  scryptParamsJSON["r"] = scryptR
  scryptParamsJSON["p"] = scryptP
  scryptParamsJSON["dklen"] = scryptDKLen
  scryptParamsJSON["salt"] = hex.EncodeToString(salt)

  cipherParamsJSON := cipherParamsJSON{
    IV: hex.EncodeToString(iv),
  }

  cryptoStruct := cryptoJSON{
    Cipher:       "aes-128-ctr",
    CipherText:   hex.EncodeToString(cipherText),
    CipherParams: cipherParamsJSON,
    KDF:          keyHeaderKDF,
    KDFParams:    scryptParamsJSON,
    MAC:          hex.EncodeToString(mac),
  }
  encryptedKeyJSON := encryptedKeyJSON{
    hex.EncodeToString(key.Address[:]),
    cryptoStruct,
    key.Id.String(),
    version,
  }
  return json.Marshal(encryptedKeyJSON)
}

func DecryptKey(keyjson []byte, auth string) (*Key, error) {
  m := make(map[string]interface{})
  if err := json.Unmarshal(keyjson, &m); err != nil {
    return nil, err
  }

  k := new(encryptedKeyJSON)
  if err := json.Unmarshal(keyjson, k); err != nil {
    return nil, err
  }
  keyBytes, keyId, err := decryptKey(k, auth)
  if err != nil {
    return nil, err
  }
  key := crypto.ToECDSAUnsafe(keyBytes)

  return &Key{
    Id:         uuid.UUID(keyId),
    Address:    crypto.PubkeyToAddress(key.PublicKey),
    PrivateKey: key,
  }, nil
}

func decryptKey(keyProtected *encryptedKeyJSON, auth string) (keyBytes []byte, keyId []byte, err error) {
  if keyProtected.Version != version {
    return nil, nil, fmt.Errorf("Version not supported: %v", keyProtected.Version)
  }

  if keyProtected.Crypto.Cipher != "aes-128-ctr" {
    return nil, nil, fmt.Errorf("Cipher not supported: %v", keyProtected.Crypto.Cipher)
  }

  keyId = uuid.Parse(keyProtected.Id)
  mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
  if err != nil {
    return nil, nil, err
  }

  iv, err := hex.DecodeString(keyProtected.Crypto.CipherParams.IV)
  if err != nil {
    return nil, nil, err
  }

  cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
  if err != nil {
    return nil, nil, err
  }

  derivedKey, err := getKDFKey(keyProtected.Crypto, auth)
  if err != nil {
    return nil, nil, err
  }

  calculatedMAC := crypto.Sha3256(derivedKey[16:32], cipherText)
  if !bytes.Equal(calculatedMAC, mac) {
    return nil, nil, ErrDecrypt
  }

  plainText, err := crypto.AESCTRXOR(derivedKey[:16], cipherText, iv)
  if err != nil {
    return nil, nil, err
  }
  return plainText, keyId, err
}

func getKDFKey(cryptoJSON cryptoJSON, auth string) ([]byte, error) {
  authArray := []byte(auth)
  salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
  if err != nil {
    return nil, err
  }
  dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])

  if cryptoJSON.KDF == keyHeaderKDF {
    n := ensureInt(cryptoJSON.KDFParams["n"])
    r := ensureInt(cryptoJSON.KDFParams["r"])
    p := ensureInt(cryptoJSON.KDFParams["p"])
    return scrypt.Key(authArray, salt, n, r, p, dkLen)
  }

  return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

// FROM GO-ETHEREUM
// TODO: can we do without this when unmarshalling dynamic JSON?
// why do integers in KDF params end up as float64 and not int after
// unmarshal?
func ensureInt(x interface{}) int {
  res, ok := x.(int)
  if !ok {
    res = int(x.(float64))
  }
  return res
}

func (ks *KeyStore) GetKey(a common.Address, auth string) (*Key, error) {
  if !ks.HasAddress(a) {
    return nil, ErrNoMatch
  }

  key, err := DecryptKey(ks.keys[a], auth)
  return key, err
}
