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

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestNewGenesisBlock(t *testing.T) {
	genesisBlock, dynasties, _ := test.NewTestGenesisBlock(t)

	assert.True(t, core.CheckGenesisBlock(genesisBlock))
	txs := genesisBlock.Transactions()
	initialMessage := "Genesis block of MediBloc"
	assert.Equalf(t, string(txs[0].Data()), initialMessage, "Initial tx payload should equal '%s'", initialMessage)

	sto := genesisBlock.Storage()

	accStateBatch, err := core.NewAccountStateBatch(genesisBlock.AccountsRoot().Bytes(), sto)
	assert.NoError(t, err)
	accState := accStateBatch.AccountState()

	addr := dynasties[0].Addr
	assert.NoError(t, err)
	acc, err := accState.GetAccount(addr.Bytes())
	assert.NoError(t, err)

	expectedBalance, _ := util.NewUint128FromString("1000000000")
	assert.Zerof(t, acc.Balance().Cmp(expectedBalance), "Balance of new account in genesis block should equal to %s", expectedBalance.String())
}
