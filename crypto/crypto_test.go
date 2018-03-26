package crypto

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
var testAddrHex = "39de21d6905bebd5b76371170b7097b85bd3bc48"

func TestSha3(t *testing.T) {
	in_a := make([]byte, 5)
	in_b := make([]byte, 8)
	ha := Sha3256(in_a)
	hb := Sha3256(in_b)

	assert.Len(t, ha, 32, "The lenght hash output should be 32 bytes")
	assert.Len(t, hb, 32, "The lenght hash output should be 32 bytes")
	assert.NotEqual(t, ha, hb, "Hash of zero-bytes with different length should not equal")
}

func TestToECDSAErrors(t *testing.T) {
	_, err := HexToECDSA("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NotNil(t, err, "HexToECDSA should've returned error")

	_, err = HexToECDSA("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	assert.NotNil(t, err, "HexToECDSA should've returned error")
}

func TestSign(t *testing.T) {
	assert := assert.New(t)
	key, _ := HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)

	msg := Sha3256([]byte("foo"))
	sig, err := Sign(msg, key)
	assert.NoErrorf(err, "Sign error: %s", err)

	recoveredPub, err := Ecrecover(msg, sig)
	assert.NoErrorf(err, "ECRecover error: %s", err)

	pubKey := ToECDSAPub(recoveredPub)
	recoveredAddr := PubkeyToAddress(*pubKey)
	assert.Equalf(addr, recoveredAddr, "Address mismatch: want: %x have: %x", addr, recoveredAddr)

	// should be equal to SigToPub
	recoveredPub2, err := SigToPub(msg, sig)
	assert.NoErrorf(err, "ECRecover error: %s", err)

	recoveredAddr2 := PubkeyToAddress(*recoveredPub2)
	assert.Equalf(addr, recoveredAddr2, "Address mismatch: want: %x have: %x", addr, recoveredAddr2)
}

func TestInvalidSign(t *testing.T) {
	_, err := Sign(make([]byte, 1), nil)
	assert.Error(t, err, "expected sign with hash 1 byte to error")

	_, err = Sign(make([]byte, 33), nil)
	assert.Error(t, err, "expected sign with hash 33 byte to error")
}
