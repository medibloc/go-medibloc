package keystore

import (
  "crypto/ecdsa"
  "encoding/hex"
  "encoding/json"
  "fmt"
  "github.com/medibloc/go-medibloc/accounts"
  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/crypto"
  "github.com/pborman/uuid"
  "io"
  "io/ioutil"
  "os"
  "path/filepath"
  "time"
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

func storeNewKey(ks *KeyStore, rand io.Reader, auth string) (*Key, accounts.Account, error) {
  key, err := newKey(rand)
  if err != nil {
    return nil, accounts.Account{}, err
  }
  a := accounts.Account{Address: key.Address, URL: accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(keyFileName(key.Address))}}
  if err := ks.StoreKey(a.URL.Path, key, auth); err != nil {
    zeroKey(key.PrivateKey)
    return nil, a, err
  }
  return key, a, err
}

func writeKeyFile(file string, content []byte) error {
  // Create the keystore directory with appropriate permissions
  // in case it is not present yet.
  const dirPerm = 0700
  if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
    return err
  }
  // Atomic write: create a temporary hidden file first
  // then move it into place. TempFile assigns mode 0600.
  f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
  if err != nil {
    return err
  }
  if _, err := f.Write(content); err != nil {
    f.Close()
    os.Remove(f.Name())
    return err
  }
  f.Close()
  return os.Rename(f.Name(), file)
}

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyAddr common.Address) string {
  ts := time.Now().UTC()
  return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
  var tz string
  name, offset := t.Zone()
  if name == "UTC" {
    tz = "Z"
  } else {
    tz = fmt.Sprintf("%03d00", offset/3600)
  }
  return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
  b := k.D.Bits()
  for i := range b {
    b[i] = 0
  }
}
