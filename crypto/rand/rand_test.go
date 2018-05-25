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
