package crypto

import (
	"crypto/ecdsa"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/common/hexutil"
	"github.com/medibloc/go-medibloc/common/math"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testmsg     = hexutil.MustDecode("0xce0677bb30baa8cf067c88db9811f4333d131bf8bcf12fe7065d211dce971008")
	testsig     = hexutil.MustDecode("0x90f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e549984a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc9301")
	testpubkey  = hexutil.MustDecode("0x04e32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a0a2b2667f7e725ceea70c673093bf67663e0312623c8e091b13cf2c0f11ef652")
	testpubkeyc = hexutil.MustDecode("0x02e32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a")
)

func TestEcrecover(t *testing.T) {
	pubkey, err := Ecrecover(testmsg, testsig)
	assert.NoErrorf(t, err, "recover error: %s", err)
	assert.Equalf(t, pubkey, testpubkey, "pubkey mismatch: want: %x have: %x", testpubkey, pubkey)
}

func TestVerifySignature(t *testing.T) {
	assert := assert.New(t)
	sig := testsig[:len(testsig)-1] // remove recovery id
	assert.True(VerifySignature(testpubkey, testmsg, sig), "can't verify signature with uncompressed key")
	assert.True(VerifySignature(testpubkeyc, testmsg, sig), "can't verify signature with compressed key")

	assert.False(VerifySignature(nil, testmsg, sig), "signature valid with no key")
	assert.False(VerifySignature(testpubkey, nil, sig), "signature valid with no message")
	assert.False(VerifySignature(testpubkey, testmsg, nil), "nil signature valid")
	assert.False(VerifySignature(testpubkey, testmsg, append(common.CopyBytes(sig), 1, 2, 3)), "signature valid with extra bytes at the end")
	assert.False(VerifySignature(testpubkey, testmsg, sig[:len(sig)-2]), "signature valid even though it's incomplete")

	wrongkey := common.CopyBytes(testpubkey)
	wrongkey[10]++
	assert.False(VerifySignature(wrongkey, testmsg, sig), "signature valid with with wrong public key")
}

// This test checks that VerifySignature rejects malleable signatures with s > N/2.
func TestVerifySignatureMalleable(t *testing.T) {
	sig := hexutil.MustDecode("0x638a54215d80a6713c8d523a6adc4e6e73652d859103a36b700851cb0e61b66b8ebfc1a610c57d732ec6e0a8f06a9a7a28df5051ece514702ff9cdff0b11f454")
	key := hexutil.MustDecode("0x03ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd3138")
	msg := hexutil.MustDecode("0xd301ce462d3e639518f482c7f03821fec1e602018630ce621e1e7851c12343a6")
	if VerifySignature(key, msg, sig) {
		t.Error("VerifySignature returned true for malleable signature")
	}
}

func TestDecompressPubkey(t *testing.T) {
	assert := assert.New(t)
	key, err := DecompressPubkey(testpubkeyc)
	assert.NoError(err)

	uncompressed := FromECDSAPub(key)
	assert.Equalf(uncompressed, testpubkey, "wrong public key result: got %x, want %x", uncompressed, testpubkey)

	_, err = DecompressPubkey(nil)
	assert.Error(err, "no error for nil pubkey")

	_, err = DecompressPubkey(testpubkeyc[:5])
	assert.Error(err, "no error for incomplete pubkey")

	_, err = DecompressPubkey(append(common.CopyBytes(testpubkeyc), 1, 2, 3))
	assert.Error(err, "no error for pubkey with extra bytes at the end")
}

func TestCompressPubkey(t *testing.T) {
	key := &ecdsa.PublicKey{
		Curve: S256(),
		X:     math.MustParseBig256("0xe32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a"),
		Y:     math.MustParseBig256("0x0a2b2667f7e725ceea70c673093bf67663e0312623c8e091b13cf2c0f11ef652"),
	}
	compressed := CompressPubkey(key)
	assert.Equalf(t, compressed, testpubkeyc, "wrong public key result: got %x, want %x", compressed, testpubkeyc)
}

func TestPubkeyRandom(t *testing.T) {
	const runs = 200

	for i := 0; i < runs; i++ {
		key, err := GenerateKey()
		assert.NoErrorf(t, err, "iteration %d: %v", i, err)

		pubkey2, err := DecompressPubkey(CompressPubkey(&key.PublicKey))
		assert.NoErrorf(t, err, "iteration %d: %v", i, err)
		assert.Equalf(t, key.PublicKey, *pubkey2, "iteration %d: keys not equal", i)
	}
}
