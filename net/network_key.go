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
	"path"

	"github.com/medibloc/go-medibloc/medlet/pb"

	libcrypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// LoadNetworkKeyFromFile load network priv key from file.
func LoadNetworkKeyFromFile(path string) (libcrypto.PrivKey, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return UnmarshalNetworkKey(string(data))
}

// LoadOrNewNetworkKey load network priv key from file or create new one.
func LoadOrNewNetworkKey(cfg *medletpb.Config) (libcrypto.PrivKey, error) {
	// try receiver load from conf
	key, err := LoadNetworkKeyFromFile(cfg.Network.NetworkKeyFile)
	if err != nil && !os.IsNotExist(err) {
		logging.Console().WithFields(logrus.Fields{
			"err":     err,
			"keyFile": cfg.Network.NetworkKeyFile,
			"tt":      os.IsNotExist(err),
		}).Error("No network key files")
		return nil, err
	}
	if err == nil {
		logging.Console().WithFields(logrus.Fields{
			"privateKeyPath": cfg.Network.NetworkKeyFile,
		}).Info("Load Network Key from cache file")
		return key, nil
	}

	// try receiver load from cache
	cacheDir := path.Join(cfg.Global.Datadir, DefaultPrivateKeyFile)
	key, err = LoadNetworkKeyFromFile(cacheDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		logging.Console().WithFields(logrus.Fields{
			"privateKeyPath": cacheDir,
		}).Info("Load Network Key from cache file")
		return key, nil
	}

	// generate new key and save on cache
	key, _, err = libcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	keyString, _ := MarshalNetworkKey(key)

	err = os.MkdirAll(cfg.Global.Datadir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(cacheDir, []byte(keyString), 400)
	if err != nil {
		return nil, err
	}
	logging.Console().WithFields(logrus.Fields{
		"privateKeyPath": cacheDir,
	}).Info("New network Key is generated")

	return key, nil
}

// UnmarshalNetworkKey unmarshal network key.
func UnmarshalNetworkKey(data string) (libcrypto.PrivKey, error) {
	binaryData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	return libcrypto.UnmarshalPrivateKey(binaryData)
}

// MarshalNetworkKey marshal network key.
func MarshalNetworkKey(key libcrypto.PrivKey) (string, error) {
	binaryData, err := libcrypto.MarshalPrivateKey(key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(binaryData), nil
}

// GenerateEd25519Key return a new generated Ed22519 Private key.
func GenerateEd25519Key() (libcrypto.PrivKey, error) {
	key, _, err := libcrypto.GenerateEd25519Key(rand.Reader)
	return key, err
}
