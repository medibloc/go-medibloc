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

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/mitchellh/copystructure"
	"github.com/stretchr/testify/require"
)

const (
	defaultSignAlg = algorithm.SECP256K1
)

// BlockBuilder builds block.
type BlockBuilder struct {
	t  *testing.T
	pb *corepb.Block
}

// NewBlockBuilder creates BlockBuilder.
func NewBlockBuilder(t *testing.T) *BlockBuilder {
	return &BlockBuilder{
		t:  t,
		pb: &corepb.Block{},
	}
}

// NewBlockBuilderFrom creates BlockBuilder from existing block data.
func NewBlockBuilderFrom(t *testing.T, bd *core.BlockData) *BlockBuilder {
	pb, err := bd.ToProto()
	require.NoError(t, err)
	return &BlockBuilder{
		t:  t,
		pb: pb.(*corepb.Block),
	}
}

// Hash returns hash.
func (bb *BlockBuilder) Hash() []byte {
	return bb.pb.Header.Hash
}

// SetHash sets hash.
func (bb *BlockBuilder) SetHash(hash []byte) *BlockBuilder {
	bb.pb.Header.Hash = hash
	return bb
}

// CalcHash calculates and sets hash.
func (bb *BlockBuilder) CalcHash() *BlockBuilder {
	var bd core.BlockData

	err := bd.FromProto(bb.pb)
	require.NoError(bb.t, err)

	hash := core.HashBlockData(&bd)
	bb.pb.Header.Hash = hash
	return bb
}

// ParentHash return parent's hash.
func (bb *BlockBuilder) ParentHash() []byte {
	return bb.pb.Header.ParentHash
}

// SetParentHash sets parent's hash.
func (bb *BlockBuilder) SetParentHash(hash []byte) *BlockBuilder {
	bb.pb.Header.ParentHash = hash
	return bb
}

// AccountRoot returns account root.
func (bb *BlockBuilder) AccountRoot() []byte {
	return bb.pb.Header.AccsRoot
}

// SetAccountRoot sets account root.
func (bb *BlockBuilder) SetAccountRoot(root []byte) *BlockBuilder {
	bb.pb.Header.AccsRoot = root
	return bb
}

// TransactionRoot returns transaction root.
func (bb *BlockBuilder) TransactionRoot() []byte {
	return bb.pb.Header.TxsRoot
}

// SetTransactionRoot sets transaction root.
func (bb *BlockBuilder) SetTransactionRoot(root []byte) *BlockBuilder {
	bb.pb.Header.TxsRoot = root
	return bb
}

// UsageRoot  returns usage root.
func (bb *BlockBuilder) UsageRoot() []byte {
	return bb.pb.Header.UsageRoot
}

// SetUsageRoot sets usage root.
func (bb *BlockBuilder) SetUsageRoot(root []byte) *BlockBuilder {
	bb.pb.Header.UsageRoot = root
	return bb
}

// RecordRoot returns record root.
func (bb *BlockBuilder) RecordRoot() []byte {
	return bb.pb.Header.RecordsRoot
}

// SetRecordRoot sets record root.
func (bb *BlockBuilder) SetRecordRoot(root []byte) *BlockBuilder {
	bb.pb.Header.RecordsRoot = root
	return bb
}

// CandidateRoot returns candidate root.
func (bb *BlockBuilder) CandidateRoot() []byte {
	return bb.pb.Header.CandidacyRoot
}

// SetCandidateRoot sets candidate root.
func (bb *BlockBuilder) SetCandidateRoot(root []byte) *BlockBuilder {
	bb.pb.Header.CandidacyRoot = root
	return bb
}

// CertificateRoot returns certificate root.
func (bb *BlockBuilder) CertificateRoot() []byte {
	return bb.pb.Header.CertificationRoot
}

// SetCertificateRoot sets certificate root.
func (bb *BlockBuilder) SetCertificateRoot(root []byte) *BlockBuilder {
	bb.pb.Header.CertificationRoot = root
	return bb
}

// ConsensusRoot returns consensus root.
func (bb *BlockBuilder) ConsensusRoot() []byte {
	return bb.pb.Header.ConsensusRoot
}

// SetConsensusRoot sets consensus root.
func (bb *BlockBuilder) SetConsensusRoot(root []byte) *BlockBuilder {
	bb.pb.Header.ConsensusRoot = root
	return bb
}

// ReservationQueueRoot returns reservation queue root.
func (bb *BlockBuilder) ReservationQueueRoot() []byte {
	return bb.pb.Header.ReservationQueueHash
}

// SetReservationQueueRoot sets reservation queue root.
func (bb *BlockBuilder) SetReservationQueueRoot(root []byte) *BlockBuilder {
	bb.pb.Header.ReservationQueueHash = root
	return bb
}

// Coinbase returns coinbase.
func (bb *BlockBuilder) Coinbase() common.Address {
	return common.BytesToAddress(bb.pb.Header.Coinbase)
}

// SetCoinbase sets coinbase.
func (bb *BlockBuilder) SetCoinbase(addr common.Address) *BlockBuilder {
	bb.pb.Header.Coinbase = addr.Bytes()
	return bb
}

// Timestamp returns timestamp.
func (bb *BlockBuilder) Timestamp() int64 {
	return bb.pb.Header.Timestamp
}

// SetTimestamp sets timestamp.
func (bb *BlockBuilder) SetTimestamp(ts int64) *BlockBuilder {
	bb.pb.Header.Timestamp = ts
	return bb
}

// ChainID returns chain ID.
func (bb *BlockBuilder) ChainID() uint32 {
	return bb.pb.Header.ChainId
}

// SetChainID sets chain ID.
func (bb *BlockBuilder) SetChainID(chainID uint32) *BlockBuilder {
	bb.pb.Header.ChainId = chainID
	return bb
}

// Alg returns crypto algorithm used for signature.
func (bb *BlockBuilder) Alg() algorithm.Algorithm {
	return algorithm.Algorithm(bb.pb.Header.Alg)
}

// SetAlg sets crypto algorithm.
func (bb *BlockBuilder) SetAlg(alg algorithm.Algorithm) *BlockBuilder {
	bb.pb.Header.Alg = uint32(alg)
	return bb
}

// Signature returns signature.
func (bb *BlockBuilder) Signature() []byte {
	return bb.pb.Header.Sign
}

// SetSignature sets signature.
func (bb *BlockBuilder) SetSignature(sign []byte) *BlockBuilder {
	bb.pb.Header.Sign = sign
	return bb
}

// Sign generates and sets block's signature.
func (bb *BlockBuilder) Sign(key signature.PrivateKey) *BlockBuilder {
	require.NotNil(bb.t, bb.pb.Header.Hash)

	signer, err := crypto.NewSignature(defaultSignAlg)
	require.NoError(bb.t, err)
	signer.InitSign(key)

	bb.pb.Header.Alg = uint32(signer.Algorithm())

	sig, err := signer.Sign(bb.pb.Header.Hash)
	require.NoError(bb.t, err)
	bb.pb.Header.Sign = sig

	return bb
}

// Transactions returns transactions.
func (bb *BlockBuilder) Transactions() []*corepb.Transaction {
	return bb.pb.Transactions
}

// SetTransactions sets transactions.
func (bb *BlockBuilder) SetTransactions(txs []*corepb.Transaction) *BlockBuilder {
	bb.pb.Transactions = txs
	return bb
}

// Height returns block height.
func (bb *BlockBuilder) Height() uint64 {
	return bb.pb.Height
}

// SetHeight sets block height.
func (bb *BlockBuilder) SetHeight(height uint64) *BlockBuilder {
	bb.pb.Height = height
	return bb
}

// Build builds block.
func (bb *BlockBuilder) Build() *core.BlockData {
	var bd core.BlockData
	err := bd.FromProto(bb.pb)
	require.NoError(bb.t, err)
	return &bd
}

// BuildProto builds block in protobuf format.
func (bb *BlockBuilder) BuildProto() *corepb.Block {
	pb, err := copystructure.Copy(bb.pb)
	require.NoError(bb.t, err)
	return pb.(*corepb.Block)
}

// BuildBytes builds block in bytes.
func (bb *BlockBuilder) BuildBytes() []byte {
	data, err := proto.Marshal(bb.pb)
	require.NoError(bb.t, err)
	return data
}
