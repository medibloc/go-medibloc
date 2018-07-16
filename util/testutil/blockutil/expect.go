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
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package blockutil

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/stretchr/testify/require"
)

type Expect struct {
	t     *testing.T
	block *core.Block
}

func NewExpect(t *testing.T, block *core.Block) *Expect {
	return &Expect{
		t:     t,
		block: block,
	}
}

func (e *Expect) account(addr common.Address) core.Account {
	acc, err := e.block.State().GetAccount(addr)
	require.NoError(e.t, err)
	return acc
}

func (e *Expect) Balance(addr common.Address, value uint64) *Expect {
	acc := e.account(addr)
	require.Equal(e.t, acc.Balance().Uint64(), value)
	return e
}

func (e *Expect) Vesting(addr common.Address, vest uint64) *Expect {
	acc := e.account(addr)
	require.Equal(e.t, acc.Vesting().Uint64(), vest)
	return e
}

func (e *Expect) Nonce(addr common.Address, nonce uint64) *Expect {
	acc := e.account(addr)
	require.Equal(e.t, acc.Nonce(), nonce)
	return e
}
