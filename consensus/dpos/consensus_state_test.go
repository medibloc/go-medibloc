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

package dpos_test

import (
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/assert"
)

func TestLoadConsensusState(t *testing.T) {
	genesis, _, _ := testutil.NewTestGenesisBlock(t, 21)
	cs, err := dpos.NewConsensusState(nil, genesis.Storage())
	assert.NoError(t, err)

	time.Sleep(10)

	root1, err := cs.RootBytes()
	newCs, err := dpos.LoadConsensusState(root1, genesis.Storage())
	assert.NoError(t, err)
	root2, err := newCs.RootBytes()
	assert.NoError(t, err)
	assert.Equal(t, root1, root2)
}

func TestClone(t *testing.T) {
	genesis, _, _ := testutil.NewTestGenesisBlock(t, 21)
	cs, err := dpos.NewConsensusState(nil, genesis.Storage())
	assert.NoError(t, err)

	clone, err := cs.Clone()
	assert.NoError(t, err)
	assert.Equal(t, cs, clone)
}
