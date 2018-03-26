package keystore

import (
  "crypto/ecdsa"
  "encoding/hex"
  "encoding/json"
  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/crypto"
  "github.com/pborman/uuid"
  "io"
)

const (
  // version compatible with ethereum, the version start with 3
  version = 3
)

// Algorithm type alias
type Algorithm uint8

const (
  // SECP256K1 a type of signer
  SECP256K1 Algorithm = 1
)

type Key struct {
  Id         uuid.UUID
  Address    common.Address
  PrivateKey *ecdsa.PrivateKey
}

type plainKeyJSON struct {
  Address    string `json:"address"`
  PrivateKey string `json:"privatekey"`
  Id         string `json:"id"`
  Version    int    `json:"version"`
}

type encryptedKeyJSON struct {
  Address string     `json:"address"`
  Crypto  cryptoJSON `json:"crypto"`
  Id      string     `json:"id"`
  Version int        `json:"version"`
}

type cryptoJSON struct {
  Cipher       string                 `json:"cipher"`
  CipherText   string                 `json:"ciphertext"`
  CipherParams cipherParamsJSON       `json:"cipherparams"`
  KDF          string                 `json:"kdf"`
  KDFParams    map[string]interface{} `json:"kdfparams"`
  MAC          string                 `json:"mac"`
}

type cipherParamsJSON struct {
  IV string `json:"iv"`
}

func (k *Key) MarshalJSON() ([]byte, error) {
  jStruct := plainKeyJSON{
    hex.EncodeToString(k.Address[:]),
    hex.EncodeToString(crypto.FromECDSA(k.PrivateKey)),
    k.Id.String(),
    version,
  }
  return json.Marshal(jStruct)
}

func (k *Key) UnmarshalJSON(j []byte) (err error) {
  keyJSON := new(plainKeyJSON)
  err = json.Unmarshal(j, &keyJSON)
  if err != nil {
    return err
  }

  u := new(uuid.UUID)
  *u = uuid.Parse(keyJSON.Id)
  k.Id = *u
  addr, err := hex.DecodeString(keyJSON.Address)
  if err != nil {
    return err
  }
  privKey, err := crypto.HexToECDSA(keyJSON.PrivateKey)
  if err != nil {
    return err
  }

  k.Address = common.BytesToAddress(addr)
  k.PrivateKey = privKey

  return nil
}

func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *Key {
  id := uuid.NewRandom()
  key := &Key{
    Id:         id,
    Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
    PrivateKey: privateKeyECDSA,
  }
  return key
}

func newKey(rand io.Reader) (*Key, error) {
  privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
  if err != nil {
    return nil, err
  }
  return newKeyFromECDSA(privateKeyECDSA), nil
}
