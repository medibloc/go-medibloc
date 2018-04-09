package secp256k1_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	byteutils "github.com/medibloc/go-medibloc/util/bytes"
	"github.com/stretchr/testify/assert"
)

var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
var testAddrHex = "037db227d7094ce215c3a0f57e1bcc732551fe351f94249471934567e0f5dc1bf7"

func TestToECDSAErrors(t *testing.T) {
	_, err := secp256k1.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NotNil(t, err, "HexToECDSA should've returned error")

	_, err = secp256k1.HexToECDSA("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	assert.NotNil(t, err, "HexToECDSA should've returned error")
}

func TestSign(t *testing.T) {
	assert := assert.New(t)
	key, _ := secp256k1.HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)

	msg := byteutils.Hex2Bytes("39de21d6905bebd5b76371170b7097b85bd3bc48b76371170b7097b85bd3bc48")
	sig, err := secp256k1.Sign(msg, secp256k1.FromECDSAPrivateKey(key))
	assert.NoErrorf(err, "Sign error: %s", err)

	recoveredPub, err := secp256k1.RecoverPubkey(msg, sig)
	assert.NoErrorf(err, "ECRecover error: %s", err)

	pubKey, err := secp256k1.ToECDSAPublicKey(recoveredPub)
	assert.NoError(err)
	recoveredAddr, err := common.PublicKeyToAddress(secp256k1.NewPublicKey(*pubKey))
	assert.NoError(err)
	assert.Equalf(addr, recoveredAddr, "Address mismatch: want: %x have: %x", addr, recoveredAddr)

	// should be equal to SigToPub
	recoveredPub2, err := secp256k1.RecoverPubkey(msg, sig)
	assert.NoErrorf(err, "ECRecover error: %s", err)

	recoveredPubKey2, err := secp256k1.ToECDSAPublicKey(recoveredPub2)
	assert.NoErrorf(err, "ToECDSAPublicKey error: %s", err)

	recoveredAddr2, err := common.PublicKeyToAddress(secp256k1.NewPublicKey(*recoveredPubKey2))
	assert.NoError(err)
	assert.Equalf(addr, recoveredAddr2, "Address mismatch: want: %x have: %x", addr, recoveredAddr2)
}

func TestInvalidSign(t *testing.T) {
	_, err := secp256k1.Sign(make([]byte, 1), nil)
	assert.Error(t, err, "expected sign with hash 1 byte to error")

	_, err = secp256k1.Sign(make([]byte, 33), nil)
	assert.Error(t, err, "expected sign with hash 33 byte to error")
}
