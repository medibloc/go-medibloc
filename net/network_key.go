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

package net

import (
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"

	"os"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
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
func LoadNetworkKeyFromFileOrCreateNew(cfg *Config) (crypto.PrivKey, error) {
	if cfg.PrivateKeyPath == "" {
		if _, err := os.Stat(cfg.PrivateKeyCachePath); err == nil {
			logging.Console().WithFields(logrus.Fields{
				"privateKeyPath": cfg.PrivateKeyCachePath,
			}).Info("Load Network Key from cache file")
			return LoadNetworkKeyFromFile(cfg.PrivateKeyCachePath)
		}
		key, err := GenerateEd25519Key()
		if err != nil {
			return nil, err
		}
		if cfg.PrivateKeyCachePath != "" {
			keyString, _ := MarshalNetworkKey(key)
			err = ioutil.WriteFile(cfg.PrivateKeyCachePath, []byte(keyString), 400)
			if err != nil {
				return nil, err
			}
			logging.Console().WithFields(logrus.Fields{
				"privateKeyPath": cfg.PrivateKeyCachePath,
			}).Info("Generate New Network Key")
		}
		return key, nil
	}
	logging.Console().WithFields(logrus.Fields{
		"privateKeyPath": cfg.PrivateKeyPath,
	}).Info("Load Network Key from file")
	return LoadNetworkKeyFromFile(cfg.PrivateKeyPath)
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
