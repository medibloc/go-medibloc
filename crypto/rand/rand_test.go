package rand

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandom(t *testing.T) {
	a := rand.GetEntropyCSPRNG(32)
	b := rand.GetEntropyCSPRNG(32)
	assert.NotEqual(t, a, b, "Two random outputs should not equal")
}
