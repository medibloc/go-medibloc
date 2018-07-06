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

package txutil

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

const (
	defaultSignAlg = algorithm.SECP256K1
)

// TransactionBuilder builds transaction.
type TransactionBuilder struct {
	t  *testing.T
	pb *corepb.Transaction
}

// NewTransactionBuilder creates TransactionBuilder.
func NewTransactionBuilder(t *testing.T) *TransactionBuilder {
	return &TransactionBuilder{
		t:  t,
		pb: &corepb.Transaction{},
	}
}

// NewTransactionBuilderFrom creates TransactionBuilder from existing transaction.
func NewTransactionBuilderFrom(t *testing.T, tx *core.Transaction) *TransactionBuilder {
	pb, err := tx.ToProto()
	require.NoError(t, err)
	return &TransactionBuilder{
		t:  t,
		pb: pb.(*corepb.Transaction),
	}
}

// Hash returns hash.
func (tb *TransactionBuilder) Hash() []byte {
	return tb.pb.Hash
}

// SetHash sets hash.
func (tb *TransactionBuilder) SetHash(hash []byte) *TransactionBuilder {
	tb.pb.Hash = hash
	return tb
}

// CalcHash sets calculated hash.
func (tb *TransactionBuilder) CalcHash() *TransactionBuilder {
	var tx core.Transaction
	err := tx.FromProto(tb.pb)
	require.NoError(tb.t, err)

	hash, err := tx.CalcHash()
	require.NoError(tb.t, err)
	tb.pb.Hash = hash
	return tb
}

// From returns tx's from address.
func (tb *TransactionBuilder) From() common.Address {
	return common.BytesToAddress(tb.pb.From)
}

// SetFrom sets tx's from address.
func (tb *TransactionBuilder) SetFrom(from common.Address) *TransactionBuilder {
	tb.pb.From = from.Bytes()
	return tb
}

// To returns tx's to address.
func (tb *TransactionBuilder) To() common.Address {
	return common.BytesToAddress(tb.pb.To)
}

// SetTo sets tx's to address.
func (tb *TransactionBuilder) SetTo(to common.Address) *TransactionBuilder {
	tb.pb.To = to.Bytes()
	return tb
}

// Value returns tx's value.
func (tb *TransactionBuilder) Value() uint64 {
	v, err := util.NewUint128FromFixedSizeByteSlice(tb.pb.Value)
	require.NoError(tb.t, err)
	return v.Uint64()
}

// SetValue sets tx's value.
func (tb *TransactionBuilder) SetValue(value uint64) *TransactionBuilder {
	b, err := util.NewUint128FromUint(value).ToFixedSizeByteSlice()
	require.NoError(tb.t, err)
	tb.pb.Value = b
	return tb
}

// Timestamp returns tx's timestamp.
func (tb *TransactionBuilder) Timestamp() int64 {
	return tb.pb.Timestamp
}

// SetTimestamp sets tx's timestamp.
func (tb *TransactionBuilder) SetTimestamp(ts int64) *TransactionBuilder {
	tb.pb.Timestamp = ts
	return tb
}

//TODO data

// Nonce returns tx's nonce.
func (tb *TransactionBuilder) Nonce() uint64 {
	return tb.pb.Nonce
}

// SetNonce sets tx's nonce.
func (tb *TransactionBuilder) SetNonce(nonce uint64) *TransactionBuilder {
	tb.pb.Nonce = nonce
	return tb
}

// ChainID returns tx's chainID.
func (tb *TransactionBuilder) ChainID() uint32 {
	return tb.pb.ChainId
}

// SetChainID sets tx's chainID.
func (tb *TransactionBuilder) SetChainID(chainID uint32) *TransactionBuilder {
	tb.pb.ChainId = chainID
	return tb
}

// Alg returns tx's algorithm.
func (tb *TransactionBuilder) Alg() algorithm.Algorithm {
	return algorithm.Algorithm(tb.pb.Alg)
}

// SetAlg sets tx's algorithm.
func (tb *TransactionBuilder) SetAlg(alg algorithm.Algorithm) *TransactionBuilder {
	tb.pb.Alg = uint32(alg)
	return tb
}

// Signature returns tx's signature.
func (tb *TransactionBuilder) Signature() []byte {
	return tb.pb.Sign
}

// SetSignature sets tx's signature.
func (tb *TransactionBuilder) SetSignature(sign []byte) *TransactionBuilder {
	tb.pb.Sign = sign
	return tb
}

// Sign generates and sets tx's signature.
func (tb *TransactionBuilder) Sign(key signature.PrivateKey) *TransactionBuilder {
	require.NotNil(tb.t, tb.pb.Hash)

	signer, err := crypto.NewSignature(defaultSignAlg)
	require.NoError(tb.t, err)
	signer.InitSign(key)

	tb.pb.Alg = uint32(signer.Algorithm())

	sig, err := signer.Sign(tb.pb.Hash)
	require.NoError(tb.t, err)
	tb.pb.Sign = sig

	return tb
}

// PayerSignature returns tx's payer signature.
func (tb *TransactionBuilder) PayerSignature() []byte {
	return tb.pb.PayerSign
}

// SetPayerSignature returns tx's payer signature.
func (tb *TransactionBuilder) SetPayerSignature(sign []byte) *TransactionBuilder {
	tb.pb.PayerSign = sign
	return tb
}

// SignPayer generates and sets tx's payer signature.
func (tb *TransactionBuilder) SignPayer(key signature.PrivateKey) *TransactionBuilder {
	require.NotNil(tb.t, tb.pb.Hash)
	require.NotNil(tb.t, tb.pb.Sign)

	hasher := sha3.New256()
	hasher.Write(tb.pb.Hash)
	hasher.Write(tb.pb.Sign)
	hash := hasher.Sum(nil)

	signer, err := crypto.NewSignature(defaultSignAlg)
	require.NoError(tb.t, err)
	signer.InitSign(key)

	sig, err := signer.Sign(hash)
	require.NoError(tb.t, err)

	tb.pb.PayerSign = sig
	return tb
}

// Build builds transaction.
func (tb *TransactionBuilder) Build() *core.Transaction {
	var tx core.Transaction
	err := tx.FromProto(tb.pb)
	require.NoError(tb.t, err)
	return &tx
}

// BuildProto builds transaction in protobuf format.
func (tb *TransactionBuilder) BuildProto() *corepb.Transaction {
	return tb.pb
}

// BuildBytes builds transaction in bytes.
func (tb *TransactionBuilder) BuildBytes() []byte {
	data, err := proto.Marshal(tb.pb)
	require.NoError(tb.t, err)
	return data
}
