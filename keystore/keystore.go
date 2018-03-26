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
  "github.com/medibloc/go-medibloc/accounts"
  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/common/math"
  "github.com/medibloc/go-medibloc/crypto"
  "github.com/medibloc/go-medibloc/crypto/rand"
  "github.com/pborman/uuid"
  "golang.org/x/crypto/scrypt"
  "io/ioutil"
  "os"
  "path/filepath"
  "sync"
  "time"
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
  ErrLocked  = accounts.NewAuthNeededError("password or unlock")
  ErrNoMatch = errors.New("no key for given address or file")
  ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

var KeyStoreScheme = "keystore"

type KeyStore struct {
  unlocked map[common.Address]*unlocked
  cache    *accountCache
  changes  chan struct{} // Channel receiving change notifications from the cache

  keysDirPath string
  scryptN     int
  scryptP     int

  mu sync.RWMutex
}

type unlocked struct {
  *Key
  abort chan struct{}
}

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore(keydir string, scryptN, scryptP int) *KeyStore {
  keydir, _ = filepath.Abs(keydir)
  ks := &KeyStore{
    keysDirPath: keydir,
    scryptN:     scryptN,
    scryptP:     scryptP,
  }
  ks.init(keydir)
  return ks
}

func (ks *KeyStore) init(keydir string) {
  ks.mu.Lock()
  defer ks.mu.Unlock()

  ks.unlocked = make(map[common.Address]*unlocked)
  ks.cache, ks.changes = newAccountCache(keydir)
}

// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (ks *KeyStore) NewAccount(passphrase string) (accounts.Account, error) {
  _, account, err := storeNewKey(ks, crand.Reader, passphrase)
  if err != nil {
    return accounts.Account{}, err
  }
  // Add the account to the cache immediately rather
  // than waiting for file system notifications to pick it up.
  ks.cache.add(account)
  return account, nil
}

// Update changes the passphrase of an existing account.
func (ks *KeyStore) Update(a accounts.Account, passphrase, newPassphrase string) error {
  a, key, err := ks.GetKey(a, passphrase)
  if err != nil {
    return err
  }
  return ks.StoreKey(a.URL.Path, key, newPassphrase)
}

func (ks *KeyStore) Delete(a accounts.Account, passphrase string) error {
  // Decrypting the key isn't really necessary, but we do
  // it anyway to check the password and zero out the key
  // immediately afterwards.
  a, key, err := ks.GetKey(a, passphrase)
  if key != nil {
    zeroKey(key.PrivateKey)
  }
  if err != nil {
    return err
  }
  // The order is crucial here. The key is dropped from the
  // cache after the file is gone so that a reload happening in
  // between won't insert it into the cache again.
  err = os.Remove(a.URL.Path)
  if err == nil {
    ks.cache.delete(a)
  }
  return err
}

// HasAddress reports whether a key with the given address is present.
func (ks *KeyStore) HasAddress(addr common.Address) bool {
  return ks.cache.hasAddress(addr)
}

// Accounts returns all key files present in the directory.
func (ks *KeyStore) Accounts() []accounts.Account {
  return ks.cache.accounts()
}

// Unlock unlocks the given account indefinitely.
func (ks *KeyStore) Unlock(a accounts.Account, passphrase string) error {
  return ks.TimedUnlock(a, passphrase, 0)
}

// Lock removes the private key with the given address from memory.
func (ks *KeyStore) Lock(addr common.Address) error {
  ks.mu.Lock()
  if unl, found := ks.unlocked[addr]; found {
    ks.mu.Unlock()
    ks.expire(addr, unl, time.Duration(0)*time.Nanosecond)
  } else {
    ks.mu.Unlock()
  }
  return nil
}

// TimedUnlock unlocks the given account with the passphrase. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits. The account must match a unique key file.
//
// If the account address is already unlocked for a duration, TimedUnlock extends or
// shortens the active unlock timeout. If the address was previously unlocked
// indefinitely the timeout is not altered.
func (ks *KeyStore) TimedUnlock(a accounts.Account, passphrase string, timeout time.Duration) error {
  a, key, err := ks.GetKey(a, passphrase)
  if err != nil {
    return err
  }

  ks.mu.Lock()
  defer ks.mu.Unlock()
  u, found := ks.unlocked[a.Address]
  if found {
    if u.abort == nil {
      // The address was unlocked indefinitely, so unlocking
      // it with a timeout would be confusing.
      zeroKey(key.PrivateKey)
      return nil
    }
    // Terminate the expire goroutine and replace it below.
    close(u.abort)
  }
  if timeout > 0 {
    u = &unlocked{Key: key, abort: make(chan struct{})}
    go ks.expire(a.Address, u, timeout)
  } else {
    u = &unlocked{Key: key}
  }
  ks.unlocked[a.Address] = u
  return nil
}

func (ks *KeyStore) expire(addr common.Address, u *unlocked, timeout time.Duration) {
  t := time.NewTimer(timeout)
  defer t.Stop()
  select {
  case <-u.abort:
    // just quit
  case <-t.C:
    ks.mu.Lock()
    // only drop if it's still the same key instance that dropLater
    // was launched with. we can check that using pointer equality
    // because the map stores a new pointer every time the key is
    // unlocked.
    if ks.unlocked[addr] == u {
      zeroKey(u.PrivateKey)
      delete(ks.unlocked, addr)
    }
    ks.mu.Unlock()
  }
}

func (ks *KeyStore) getKeyFromFile(addr common.Address, filename, auth string) (*Key, error) {
  keyjson, err := ioutil.ReadFile(filename)
  if err != nil {
    return nil, err
  }
  key, err := DecryptKey(keyjson, auth)
  if err != nil {
    return nil, err
  }
  if key.Address != addr {
    return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
  }
  return key, nil
}

func StoreKey(dir, auth string, scryptN, scryptP int) (common.Address, error) {
  _, a, err := storeNewKey(&KeyStore{
    keysDirPath: dir,
    scryptN:     scryptN,
    scryptP:     scryptP}, crand.Reader, auth)
  return a.Address, err
}

func (ks *KeyStore) StoreKey(filename string, key *Key, auth string) error {
  keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
  if err != nil {
    return err
  }
  return writeKeyFile(filename, keyjson)
}

func (ks *KeyStore) JoinPath(filename string) string {
  if filepath.IsAbs(filename) {
    return filename
  } else {
    return filepath.Join(ks.keysDirPath, filename)
  }
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

func (ks *KeyStore) GetUnlockedKey(a accounts.Account) (*Key, error) {
  ks.mu.RLock()
  defer ks.mu.RUnlock()

  unlockedKey, found := ks.unlocked[a.Address]
  if !found {
    return nil, errors.New("unlocked key not found")
  }

  return unlockedKey.Key, nil
}

func (ks *KeyStore) Find(a accounts.Account) (accounts.Account, error) {
  ks.cache.maybeReload()
  ks.cache.mu.Lock()
  defer ks.cache.mu.Unlock()
  return ks.cache.find(a)
}

func (ks *KeyStore) GetKey(a accounts.Account, auth string) (accounts.Account, *Key, error) {
  a, err := ks.Find(a)
  if err != nil {
    return a, nil, err
  }

  key, err := ks.getKeyFromFile(a.Address, a.URL.Path, auth)
  return a, key, err
}
