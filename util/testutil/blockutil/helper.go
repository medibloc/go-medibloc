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

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/stretchr/testify/require"
)

const (
	defaultSignAlg = algorithm.SECP256K1
	dynastySize    = 3
)

var defaultTxMap = core.TxFactory{
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

func signer(t *testing.T, key signature.PrivateKey) signature.Signature {
	signer, err := crypto.NewSignature(defaultSignAlg)
	require.NoError(t, err)
	signer.InitSign(key)
	return signer
}
