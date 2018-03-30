package net

import (
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

// LoadNetworkKeyFromFile load network priv key from file.
func LoadNetworkKeyFromFile(path string) (crypto.PrivKey, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return UnmarshalNetworkKey(string(data))
}

// LoadNetworkKeyFromFileOrCreateNew load network priv key from file or create new one.
func LoadNetworkKeyFromFileOrCreateNew(path string) (crypto.PrivKey, error) {
	if path == "" {
		return GenerateEd25519Key()
	}
	return LoadNetworkKeyFromFile(path)
}

// UnmarshalNetworkKey unmarshal network key.
func UnmarshalNetworkKey(data string) (crypto.PrivKey, error) {
	binaryData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	return crypto.UnmarshalPrivateKey(binaryData)
}

// MarshalNetworkKey marshal network key.
func MarshalNetworkKey(key crypto.PrivKey) (string, error) {
	binaryData, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(binaryData), nil
}

// GenerateEd25519Key return a new generated Ed22519 Private key.
func GenerateEd25519Key() (crypto.PrivKey, error) {
	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	return key, err
}
