package crypto

import (
	"crypto/aes"
	"github.com/medibloc/go-medibloc/crypto/rand"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAESDecryption(t *testing.T) {
	key := make([]byte, 32)
	in := make([]byte, 32)
	iv := rand.GetEntropyCSPRNG(aes.BlockSize)

	out, err := AESCTRXOR(key, in, iv)
	assert.Nil(t, err)
	assert.NotEqual(t, in, out, "Input and encrypted bytes should not equal")
	dec, err := AESCTRXOR(key, out, iv)
	assert.Nil(t, err)

	assert.Equal(t, in, dec, "Input and decrypted bytes should equal")
}
