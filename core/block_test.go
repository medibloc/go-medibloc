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
	"time"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
)


func TestExecuteReservedTasks(t *testing.T) {
	newBlock.Commit()

	state := newBlock.State()

	acc, err := state.GetAccount(from)
	assert.NoError(t, err)
	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(333)))
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-333)))
	tasks := state.GetReservedTasks()
	assert.Equal(t, 3, len(tasks))
	for i := 0; i < len(tasks); i++ {
		assert.Equal(t, core.RtWithdrawType, tasks[i].TaskType())
		assert.Equal(t, from, tasks[i].From())
		assert.Equal(t, withdrawTx.Timestamp()+int64(i+1)*core.RtWithdrawInterval, tasks[i].Timestamp())
	}

	newBlock.SetTimestamp(newBlock.Timestamp() + int64(2)*core.RtWithdrawInterval)
	newBlock.BeginBatch()
	assert.NoError(t, newBlock.ExecuteReservedTasks())
	newBlock.Commit()

	acc, err = state.GetAccount(from)
	assert.NoError(t, err)
	assert.Equal(t, acc.Vesting(), util.NewUint128FromUint(uint64(111)))
	assert.Equal(t, acc.Balance(), util.NewUint128FromUint(uint64(1000000000-111)))
	tasks = state.GetReservedTasks()
	assert.Equal(t, 1, len(tasks))
	assert.Equal(t, core.RtWithdrawType, tasks[0].TaskType())
	assert.Equal(t, from, tasks[0].From())
	assert.Equal(t, withdrawTx.Timestamp()+int64(3)*core.RtWithdrawInterval, tasks[0].Timestamp())
}
