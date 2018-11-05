// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/sha3"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/pborman/uuid"
)

var (
	// ErrNoMatch address key doesn't match error.
	ErrNoMatch = errors.New("no key for given address")
)

const (
	keyHeaderKDF = "scrypt"
	scryptR      = 8
	scryptDKLen  = 32
	version      = 3
)

// KeyStore manages private keys.
type KeyStore struct {
	keys map[common.Address]signature.PrivateKey
}

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore() *KeyStore {
	ks := &KeyStore{}
	ks.keys = make(map[common.Address]signature.PrivateKey)
	return ks
}

// SetKey set key.
func (ks *KeyStore) SetKey(key signature.PrivateKey) (common.Address, error) {
	addr, err := common.PublicKeyToAddress(key.PublicKey())
	if err != nil {
		return common.Address{}, err
	}
	ks.keys[addr] = key
	return addr, nil
}

// Delete deletes key.
func (ks *KeyStore) Delete(a common.Address) error {
	if !ks.HasAddress(a) {
		return ErrNoMatch
	}
	delete(ks.keys, a)
	return nil
}

// HasAddress reports whether a key with the given address is present.
func (ks *KeyStore) HasAddress(addr common.Address) bool {
	return ks.keys[addr] != nil
}

// Accounts returns all key files present in the directory.
func (ks *KeyStore) Accounts() []common.Address {
	addresses := []common.Address{}
	for addr := range ks.keys {
		addresses = append(addresses, addr)
	}
	return addresses
}

// GetKey gets key.
func (ks *KeyStore) GetKey(a common.Address) (signature.PrivateKey, error) {
	if !ks.HasAddress(a) {
		return nil, ErrNoMatch
	}
	return ks.keys[a], nil
}

// MakeKeystoreV3 makes keystore file by version 3.
func MakeKeystoreV3(privKeyHex, passphrase string, fn string) error {
	var err error

	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	if err != nil {
		return err
	}
	if privKeyHex != "" {
		err = privKey.Decode(byteutils.Hex2Bytes(privKeyHex))
		if err != nil {
			return err
		}
	}

	addr, err := common.PublicKeyToAddress(privKey.PublicKey())
	if err != nil {
		return err
	}
	key := &Key{
		ID:         uuid.NewRandom(),
		Address:    addr,
		PrivateKey: privKey,
	}
	bytes, err := EncryptKey(key, passphrase, 8192, 1)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fn, bytes, 400)
	if err != nil {
		return err
	}
	return nil
}

// EncryptKey encrypts a key using the specified scrypt parameters into a json
func EncryptKey(key *Key, auth string, scryptN, scryptP int) ([]byte, error) {
	//keyBytes := math.PaddedBigBytes(key.PrivateKey.D, 32)
	keyBytes, err := key.PrivateKey.Encoded()
	if err != nil {
		return nil, err
	}
	cryptoStruct, err := EncryptDataV3(keyBytes, []byte(auth), scryptN, scryptP)
	if err != nil {
		return nil, err
	}
	encryptedKeyJSONV3 := EncryptedKeyJSONV3{
		Address: hex.EncodeToString(key.Address[:]),
		Crypto:  cryptoStruct,
		ID:      key.ID.String(),
		Version: 3,
	}
	return json.Marshal(encryptedKeyJSONV3)
}

// EncryptDataV3 encrypts the data given as 'data' with the password 'auth'.
func EncryptDataV3(data, auth []byte, scryptN, scryptP int) (CipherJSON, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return CipherJSON{}, err
	}
	derivedKey, err := scrypt.Key(auth, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return CipherJSON{}, err
	}
	encryptKey := derivedKey[:16]

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return CipherJSON{}, err
	}
	cipherText, err := crypto.AESCTRXOR(encryptKey, data, iv)
	if err != nil {
		return CipherJSON{}, err
	}
	macHasher := sha3.New256()
	macHasher.Write(derivedKey[16:32])
	macHasher.Write(cipherText)
	mac := macHasher.Sum(nil)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = byteutils.Bytes2Hex(salt)
	cipherParamsJSON := cipherparamsJSON{
		IV: byteutils.Bytes2Hex(iv),
	}

	cryptoStruct := CipherJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          byteutils.Bytes2Hex(mac),
	}
	return cryptoStruct, nil
}

// DecryptKey decrypts a key from a json blob, returning the private key itself.
func DecryptKey(keyjson []byte, auth string) (*Key, error) {
	// Parse the json into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(keyjson, &m); err != nil {
		return nil, err
	}
	// Depending on the version try to parse one way or another
	var (
		keyBytes, keyID []byte
		err             error
	)
	if version, ok := m["version"].(string); ok && version == "1" {
		return nil, ErrInvalidKeystoreVersion
	}
	k := new(EncryptedKeyJSONV3)
	if err := json.Unmarshal(keyjson, k); err != nil {
		return nil, ErrFailedToUnmarshalKeystoreJSON
	}
	keyBytes, keyID, err = decryptKeyV3(k, auth)
	// Handle any decryption errors and return the key
	if err != nil {
		return nil, err
	}
	keyString := byteutils.Bytes2Hex(keyBytes)
	if len(keyString) == 128 { // TODO : will be deprecated after update medjs
		keyBytes = byteutils.Hex2Bytes(numbytesToHex(keyString))
	}
	key := secp256k1.GeneratePrivateKey()
	err = key.Decode(keyBytes)
	if err != nil {
		return nil, err
	}

	addr, err := common.PublicKeyToAddress(key.PublicKey())
	if err != nil {
		return nil, err
	}
	return &Key{
		ID:         uuid.UUID(keyID),
		Address:    addr,
		PrivateKey: key,
	}, nil
}

// DecryptDataV3 decrypts CipherJSON file with auth string
func DecryptDataV3(CipherJSON CipherJSON, auth string) ([]byte, error) {
	if CipherJSON.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("cipher not supported: %v", CipherJSON.Cipher)
	}
	mac := byteutils.Hex2Bytes(CipherJSON.MAC)

	iv, err := hex.DecodeString(CipherJSON.CipherParams.IV)
	if err != nil {
		return nil, err
	}
	cipherText, err := hex.DecodeString(CipherJSON.CipherText)
	if err != nil {
		return nil, err
	}
	derivedKey, err := getKDFKey(CipherJSON, auth)
	if err != nil {
		return nil, err
	}
	macHasher := sha3.New256()
	macHasher.Write(derivedKey[16:32])
	macHasher.Write(cipherText)
	calculatedMAC := macHasher.Sum(nil)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrWrongPassphrase
	}
	plainText, err := crypto.AESCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}
	return plainText, err
}

// decryptKeyV3 decrypts encryptedKeyJSONV3 file with auth string
func decryptKeyV3(keyProtected *EncryptedKeyJSONV3, auth string) (keyBytes []byte, keyID []byte, err error) {
	if keyProtected.Version != version {
		return nil, nil, fmt.Errorf("version not supported: %v", keyProtected.Version)
	}
	keyID = uuid.Parse(keyProtected.ID)
	plainText, err := DecryptDataV3(keyProtected.Crypto, auth)
	if err != nil {
		return nil, nil, err
	}
	return plainText, keyID, nil
}

// getKDFKey key decrypt function decrypts key from crypto JSON using auth string
func getKDFKey(cryptoJSON CipherJSON, auth string) ([]byte, error) {
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

	} else if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}
	return nil, fmt.Errorf("unsupported KDF: %s", cryptoJSON.KDF)
}

// ensureInt ensures x is int
func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}

// TODO : will be deprecated after update medjs
func numbytesToHex(s string) string {
	m := make(map[string]string)
	m["30"] = "0"
	m["31"] = "1"
	m["32"] = "2"
	m["33"] = "3"
	m["34"] = "4"
	m["35"] = "5"
	m["36"] = "6"
	m["37"] = "7"
	m["38"] = "8"
	m["39"] = "9"
	m["61"] = "a"
	m["62"] = "b"
	m["63"] = "c"
	m["64"] = "d"
	m["65"] = "e"
	m["66"] = "f"
	var hex string
	for i := 0; i < len(s)/2; i++ {
		hex += m[s[2*i:2*i+2]]
	}
	return hex
}
