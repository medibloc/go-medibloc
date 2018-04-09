package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesConversion(t *testing.T) {
	bytes := []byte{5}
	hash := BytesToHash(bytes)

	var exp Hash
	exp[31] = 5

	assert.Equalf(t, hash, exp, "expected %x got %x", exp, hash)
}

func TestIsHexAddress(t *testing.T) {
	tests := []struct {
		str string
		exp bool
	}{
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed5aaeb6053f3e94c9b9a09f3366", true},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed5aaeb6053f3e94c9b9a09f3366", true},
		{"0X5aaeb6053f3e94c9b9a09f33669435e7ef1beaed5aaeb6053f3e94c9b9a09f3366", true},
		{"0XAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", true},
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed15aaeb6053f3e94c9b9a09f3366", false},
		{"0x5aaeb6053f3e94c9b9a09f33669435e7ef1beae5aaeb6053f3e94c9b9a09f3366", false},
		{"5aaeb6053f3e94c9b9a09f33669435e7ef1beaed115aaeb6053f3e94c9b9a09f3366", false},
		{"0xxaaeb6053f3e94c9b9a09f33669435e7ef1beaedaaeb6053f3e94c9b9a09f33669", false},
	}

	for _, test := range tests {
		result := IsHexAddress(test.str)
		assert.Equalf(t, result, test.exp, "IsHexAddress(%s) == %v; expected %v",
			test.str, result, test.exp)
	}
}
