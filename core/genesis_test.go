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

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/mitchellh/copystructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenesisConf(t *testing.T) {
	conf, _, _ := testutil.NewTestGenesisConf(t, 21)
	str := proto.MarshalTextString(conf)
	t.Log(str)
}

func TestNewGenesisBlock(t *testing.T) {
	dynastySize := 21
	genesisBlock, dynasties, dist := testutil.NewTestGenesisBlock(t, dynastySize)

	assert.True(t, core.CheckGenesisBlock(genesisBlock))
	txs := genesisBlock.Transactions()
	initialMessage := "Genesis block of MediBloc"
	defaultPayload := &core.DefaultPayload{
		Message: initialMessage,
	}
	payloadBuf, err := defaultPayload.ToBytes()
	assert.NoError(t, err)
	assert.Equalf(t, txs[0].Payload(), payloadBuf, "Initial tx payload should equal '%s'", initialMessage)

	for i := 0; i < len(dist); i++ {
		assert.True(t, dist[i].Addr.Equals(txs[1+i].To()))
		assert.Equal(t, "400000000000000000000", txs[1+i].Value().String())
	}

	child := blockutil.New(t, dynastySize).Block(genesisBlock).Child().Build()
	for _, dynasty := range dynasties {
		addr := dynasty.Addr

		acc, err := child.State().GetAccount(addr)
		require.NoError(t, err)
		assert.NotNil(t, acc.CandidateID)

		inDynasty, err := child.State().DposState().InDynasty(addr)
		require.NoError(t, err)
		assert.True(t, inDynasty)
	}

	accState := genesisBlock.State().AccState()
	for _, holder := range dist[:21] {
		addr := holder.Addr
		acc, err := accState.GetAccount(addr)
		assert.NoError(t, err)

		assert.Equal(t, "298000000000000000000", acc.Balance.String())
	}
	for _, holder := range dist[21:] {
		addr := holder.Addr
		acc, err := accState.GetAccount(addr)
		assert.NoError(t, err)

		assert.Equal(t, "400000000000000000000", acc.Balance.String())
	}
}

func TestCheckGenesisBlock(t *testing.T) {
	conf, _, _ := testutil.NewTestGenesisConf(t, testutil.DynastySize)
	stor, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	consensus := dpos.New(testutil.DynastySize)
	genesis, err := core.NewGenesisBlock(conf, consensus, blockutil.DefaultTxMap, stor)
	require.NoError(t, err)

	ok := core.CheckGenesisConf(genesis, conf)
	require.True(t, ok)

	modified := copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.Meta.ChainId = 9898
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.TokenDistribution = modified.TokenDistribution[1:]
	require.False(t, core.CheckGenesisConf(genesis, modified))
}
