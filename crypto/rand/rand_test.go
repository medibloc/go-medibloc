package rand

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandom(t *testing.T) {
	a := GetEntropyCSPRNG(32)
	b := GetEntropyCSPRNG(32)
	assert.NotEqual(t, a, b, "Two random outputs should not equal")
}
