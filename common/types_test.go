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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
