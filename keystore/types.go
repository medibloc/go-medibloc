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
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/pborman/uuid"
)

// CipherJSON json format for crypto field in keystore file
type CipherJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

// cipherparamsJSON
type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// Key key struct for encryption key argument
type Key struct {
	ID uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey signature.PrivateKey
}

// EncryptedKeyJSONV3 is a struct for encrypted key
type EncryptedKeyJSONV3 struct {
	Address string     `json:"address"`
	Crypto  CipherJSON `json:"crypto"`
	ID      string     `json:"id"`
	Version int        `json:"version"`
}
