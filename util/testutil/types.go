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

package testutil

import (
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
)

var (
	//ChainID is chain id for test configuration.
	ChainID uint32 = 1

	//DynastySize is dynasty size for test configuration
	DynastySize = 3

	//TxMap is TxMap
	TxMap = core.TxFactory{
		core.TxOpSend:                core.NewSendTx,
		core.TxOpAddRecord:           core.NewAddRecordTx,
		core.TxOpVest:                core.NewVestTx,
		core.TxOpWithdrawVesting:     core.NewWithdrawVestingTx,
		core.TxOpAddCertification:    core.NewAddCertificationTx,
		core.TxOpRevokeCertification: core.NewRevokeCertificationTx,

		dpos.TxOpBecomeCandidate: dpos.NewBecomeCandidateTx,
		dpos.TxOpQuitCandidacy:   dpos.NewQuitCandidateTx,
		dpos.TxOpVote:            dpos.NewVoteTx,
	}
)
