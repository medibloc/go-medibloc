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

package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMiner(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis().Child()

	miner, err := bb.Build().Miner()
	require.Error(t, err)

	singingMiner := bb.FindMiner()
	b := bb.SignPair(singingMiner).Build()

	miner, err = b.Miner()
	require.NoError(t, err)
	assert.Equal(t, singingMiner.Addr, miner)
}
