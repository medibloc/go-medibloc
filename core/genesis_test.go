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

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/mitchellh/copystructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenesisBlock(t *testing.T) {
	genesisBlock, dynasties, _ := testutil.NewTestGenesisBlock(t, 21)

	assert.True(t, core.CheckGenesisBlock(genesisBlock))
	txs := genesisBlock.Transactions()
	initialMessage := "Genesis block of MediBloc"
	assert.Equalf(t, string(txs[0].Payload()), initialMessage, "Initial tx payload should equal '%s'", initialMessage)

	accState := genesisBlock.State().AccState()

	addr := dynasties[0].Addr
	acc, err := accState.GetAccount(addr)
	assert.NoError(t, err)

	expectedBalance, _ := util.NewUint128FromString("1000000000")
	assert.Zerof(t, acc.Balance.Cmp(expectedBalance), "Balance of new account in genesis block should equal to %s", expectedBalance.String())
}

func TestCheckGenesisBlock(t *testing.T) {
	conf, _, _ := testutil.NewTestGenesisConf(t, testutil.DynastySize)
	stor, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	consensus := dpos.New(testutil.DynastySize)
	genesis, err := core.NewGenesisBlock(conf, consensus, stor)
	require.NoError(t, err)

	ok := core.CheckGenesisConf(genesis, conf)
	require.True(t, ok)

	modified := copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.Meta.ChainId = 9898
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.Meta.DynastySize = 22
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.Consensus.Dpos.Dynasty = modified.Consensus.Dpos.Dynasty[1:]
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.Consensus.Dpos.Dynasty[0] = "Wrong Address"
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.TokenDistribution = modified.TokenDistribution[1:]
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.TokenDistribution[2].Address = "Wrong Address"
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.TokenDistribution[3].Value = "989898"
	require.False(t, core.CheckGenesisConf(genesis, modified))

	modified = copystructure.Must(copystructure.Copy(conf)).(*corepb.Genesis)
	modified.TokenDistribution[4].Value = "Wrong Value"
	require.False(t, core.CheckGenesisConf(genesis, modified))
}
