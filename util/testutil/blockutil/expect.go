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
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util"
	"github.com/stretchr/testify/require"
)

// Expect is a structure for expect
type Expect struct {
	t     *testing.T
	block *core.Block
}

// NewExpect return new expect
func NewExpect(t *testing.T, block *core.Block) *Expect {
	return &Expect{
		t:     t,
		block: block,
	}
}

func (e *Expect) account(addr common.Address) *coreState.Account {
	acc, err := e.block.State().GetAccount(addr)
	require.NoError(e.t, err)
	return acc
}

// Balance compare balance of account to expected value
func (e *Expect) Balance(addr common.Address, med float64) *Expect {
	acc := e.account(addr)
	value := FloatToUint128(e.t, med)
	require.Equal(e.t, value.String(), acc.Balance.String())
	// require.Zero(e.t, acc.Balance.Cmp(value))
	return e
}

// Staking compare staking of account to expected value
func (e *Expect) Staking(addr common.Address, med float64) *Expect {
	acc := e.account(addr)
	stake := FloatToUint128(e.t, med)
	require.Zero(e.t, acc.Staking.Cmp(stake))
	return e
}

// Unstaking compares unstaking of account to expected value
func (e *Expect) Unstaking(addr common.Address, med float64) *Expect {
	acc := e.account(addr)
	unstake := FloatToUint128(e.t, med)
	require.Zero(e.t, acc.Unstaking.Cmp(unstake))
	return e
}

// LastUnstakingTs compares last update time of unstaking
func (e *Expect) LastUnstakingTs(addr common.Address, ts int64) *Expect {
	acc := e.account(addr)
	require.Equal(e.t, acc.LastUnstakingTs, ts)
	return e
}

// GetNonce gets nonce of account to expected value
func (e *Expect) GetNonce(addr common.Address) uint64 {
	acc := e.account(addr)
	return acc.Nonce
}

// Nonce compare nonce of account to expected value
func (e *Expect) Nonce(addr common.Address, nonce uint64) *Expect {
	require.Equal(e.t, nonce, e.GetNonce(addr))
	return e
}

// Points compares points of account to expected value
func (e *Expect) Points(addr common.Address, bandwidth *util.Uint128) *Expect {
	acc := e.account(addr)
	require.Equal(e.t, bandwidth.String(), acc.Points.String())
	return e
}

// LastPointsTs compares last update time of points
func (e *Expect) LastPointsTs(addr common.Address, ts int64) *Expect {
	acc := e.account(addr)
	require.Equal(e.t, acc.LastPointsTs, ts)
	return e
}
