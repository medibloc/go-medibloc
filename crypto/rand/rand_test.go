package rand

import (
	"testing"

	"github.com/medibloc/go-medibloc/crypto/rand"
	"github.com/stretchr/testify/assert"
)

func TestRandom(t *testing.T) {
	a := rand.GetEntropyCSPRNG(32)
	b := rand.GetEntropyCSPRNG(32)
	assert.NotEqual(t, a, b, "Two random outputs should not equal")
}
