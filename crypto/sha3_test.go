package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSha3(t *testing.T) {
	in_a := make([]byte, 5)
	in_b := make([]byte, 8)
	ha := Sha3256(in_a)
	hb := Sha3256(in_b)

	assert.Len(t, ha, 32, "The lenght hash output should be 32 bytes")
	assert.Len(t, hb, 32, "The lenght hash output should be 32 bytes")
	assert.NotEqual(t, ha, hb, "Hash of zero-bytes with different length should not equal")
}
