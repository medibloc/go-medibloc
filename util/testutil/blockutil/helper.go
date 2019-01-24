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
	"math/big"
	"testing"

	"github.com/medibloc/go-medibloc/util"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/stretchr/testify/require"
)

const (
	defaultSignAlg = algorithm.SECP256K1
)

var unitMed = util.NewUint128FromUint(1000000000000)

func signer(t *testing.T, key signature.PrivateKey) signature.Signature {
	signer, err := crypto.NewSignature(defaultSignAlg)
	require.NoError(t, err)
	signer.InitSign(key)
	return signer
}

// Points returns bandwidth usage of a transaction.
func Points(t *testing.T, tx *core.Transaction, b *core.Block) *util.Uint128 {
	execTx, err := core.TxConv(tx)
	require.NoError(t, err)

	points, err := execTx.Bandwidth().CalcPoints(b.State().Price())
	require.NoError(t, err)

	return points
}

// FloatToUint128 covert float to uint128 (precision is only 1e-03)
func FloatToUint128(t *testing.T, med float64) *util.Uint128 {
	value, err := unitMed.MulWithRat(big.NewRat(int64(med*1000), 1000))
	require.NoError(t, err)
	return value
}
