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

package keystore_test

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"

	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/stretchr/testify/require"
)

func TestKeyStore(t *testing.T) {
	ks := keystore.NewKeyStore()
	key, err := crypto.GenerateKey(algorithm.SECP256K1)
	require.NoError(t, err)
	a, err := ks.SetKey(key)
	require.NoError(t, err)
	require.True(t, ks.HasAddress(a))
	err = ks.Delete(a)
	require.NoError(t, err)
	require.False(t, ks.HasAddress(a))
}

func TestMakeKeystoreV3(t *testing.T) {
	tp := "test passphrase"
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	privKeyBytes, err := privKey.Encoded()
	require.NoError(t, err)
	require.NoError(t, keystore.MakeKeystoreV3(byteutils.Bytes2Hex(privKeyBytes), tp, "testKeyfile.key"))
	defer os.Remove("testKeyfile.key")
	dat, err := ioutil.ReadFile("testKeyfile.key")
	require.NoError(t, err)
	t.Log("created: ", string(dat))
	key, err := keystore.DecryptKey(dat, tp)
	require.NoError(t, err)
	recoveredPrivKey, err := key.PrivateKey.Encoded()
	require.NoError(t, err)
	assert.True(t, byteutils.Equal(privKeyBytes, recoveredPrivKey))
	pubKey := key.PrivateKey.PublicKey()
	recoveredAddr, err := common.PublicKeyToAddress(pubKey)
	require.NoError(t, err)
	t.Log("Recovered Public", recoveredAddr)
}

func TestEncryptDecryptV3(t *testing.T) {
	ks := `{"version":3,"id":"89c5a389-fa29-41da-a6b6-4700695b20db","address":"03b7f30a8f815a4d31c711ce374b9bd491fcb354dc6d6f9d9b8fc5ed1194703edb","crypto":{"ciphertext":"9e8548ccbc26480d51bbe19d3408f255c3e1637311aaf1ddc0ea15cfcb3d61650006ec157e0c8377264df23a85e44a717cec9d65eb1d046f4ce19c6e52437ba5","cipherparams":{"iv":"85aa6fc4a8001c4b779307f61ac7191f"},"cipher":"aes-128-ctr","kdf":"scrypt","kdfparams":{"dklen":32,"salt":"ff5620b8a61767da60ad04d2773381621c840f9964bf01f95a774231f43b10fd","n":8192,"r":8,"p":1},"mac":"5c4d3ebdd3f592548a6ff5c6f49d1a15294e9eae3168be96e0e30abdbfebeb61"}}`
	tp := "aA123456"
	key, err := keystore.DecryptKey([]byte(ks), tp)
	require.NoError(t, err, "Error in DecryptKey.")
	privKeyBytes, err := key.PrivateKey.Encoded()
	pubKeyBytes, err := key.PrivateKey.PublicKey().Compressed()
	require.NoError(t, err)
	t.Log(byteutils.Bytes2Hex(privKeyBytes))
	t.Log(byteutils.Bytes2Hex(pubKeyBytes))

}
