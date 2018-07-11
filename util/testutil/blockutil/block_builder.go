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
	"github.com/medibloc/go-medibloc/core"
	"github.com/mitchellh/copystructure"
	"github.com/stretchr/testify/require"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/crypto/signature"
)

type BlockBuilder struct {
	t *testing.T
	B *core.Block

	Dynasties testutil.AddrKeyPairs
	TokenDist testutil.AddrKeyPairs
}

func New(t *testing.T) *BlockBuilder {
	return &BlockBuilder{
		t: t,
	}
}

func (bb *BlockBuilder) copy() *BlockBuilder {
	v, err := copystructure.Copy(bb)
	require.NoError(bb.t, err)
	return v.(*BlockBuilder)
}

func (bb *BlockBuilder) Genesis(size int) *BlockBuilder {
	n := bb.copy()
	genesis, dynasties, tokenDist := testutil.NewTestGenesisBlock(bb.t, size)
	n.B = genesis
	n.Dynasties = dynasties
	n.TokenDist = tokenDist
	return n
}

func (bb *BlockBuilder) Block(block *core.Block) *BlockBuilder {
	n := bb.copy()
	n.B = block
	return n
}

func (bb *BlockBuilder) Tx() *TxBuilder {
	return newTxBuilder(bb)
}

/* Setters and Getters */

func (bb *BlockBuilder) Hash(hash []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetHash(hash)
	return n
}

func (bb *BlockBuilder) ParentHash(hash []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetParentHash(hash)
	return n
}

// AccountRoot sets account root.
func (bb *BlockBuilder) AccountRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetAccsRoot(root)
	return n
}

// TransactionRoot sets transaction root.
func (bb *BlockBuilder) TransactionRoot(root []byte) *BlockBuilder{
	n := bb.copy()
	n.B.SetTxsRoot(root)
	return n
}

// UsageRoot sets usage root.
func (bb *BlockBuilder) UsageRoot(root []byte) *BlockBuilder{
	n := bb.copy()
	n.B.SetUsageRoot(root)
	return n
}

// RecordRoot sets record root.
func (bb *BlockBuilder) RecordRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetRecordsRoot(root)
	return n
}

// CertificateRoot sets certificate root.
func (bb *BlockBuilder) CertificateRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetCertificationRoot(root)
	return n
}

// DposRoot sets dpos root.
func (bb *BlockBuilder) DposRoot(root []byte) *BlockBuilder{
	n := bb.copy()
	n.B.SetDposRoot(root)
	return n
}

// ReservationQueueRoot sets reservation queue root.
func (bb *BlockBuilder) ReservationQueueRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetReservationQueueHash(root)
	return n
}

// Coinbase sets coinbase.
func (bb *BlockBuilder) Coinbase(addr common.Address) *BlockBuilder{
	n := bb.copy()
	n.B.SetCoinbase(addr)
	return n
}

// Timestamp sets timestamp.
func (bb *BlockBuilder) Timestamp(ts int64) *BlockBuilder {
	n := bb.copy()
	n.B.SetTimestamp(ts)
	return n
}

// ChainID sets chain ID.
func (bb *BlockBuilder) ChainID(chainID uint32) *BlockBuilder{
	n := bb.copy()
	n.B.SetChainID(chainID)
	return n
}

// Alg sets crypto algorithm.
func (bb *BlockBuilder) Alg(alg algorithm.Algorithm) *BlockBuilder{
	n := bb.copy()
	n.B.SetAlg(alg)
	return n
}

// Sign sets signature.
func (bb *BlockBuilder) Sign(sign []byte) *BlockBuilder{
	n := bb.copy()
	n.B.SetSign(sign)
	return n
}

// Height sets block height.
func (bb *BlockBuilder) Height(height uint64) *BlockBuilder{
	n := bb.copy()
	n.B.SetHeight(height)
	return n
}

/* Additional Commands */

func (bb *BlockBuilder) CalcHash() *BlockBuilder {
	n := bb.copy()
	hash := core.HashBlockData(n.B.GetBlockData())
	n.B.SetHash(hash)
	return n
}

func (bb *BlockBuilder) SignKey(key signature.PrivateKey) *BlockBuilder {
	n := bb.copy()
	t := bb.t

	signer := signer(t, key)

	n.B.SetAlg(signer.Algorithm())

	sig, err := signer.Sign(n.B.Hash())
	require.NoError(t, err)
	n.B.SetSign(sig)
	return n
}

func (bb *BlockBuilder) AddTx(tx *core.Transaction) *BlockBuilder {
	n := bb.copy()
	txs := n.B.Transactions()
	txs = append(txs, tx)
	n.B.SetTransactions(txs)
	return n
}

func (bb *BlockBuilder) ExecuteTx(tx *core.Transaction) *BlockBuilder {
	n := bb.copy()
	err := n.B.Execute(tx, defaultTxMap)
	require.NoError(bb.t, err)
	err = n.B.AcceptTransaction(tx)
	require.NoError(bb.t, err)
	return n
}

// Build builds block.
func (bb *BlockBuilder) Build() *core.Block {
	n := bb.copy()
	return n.B
}

// BuildProto builds block in protobuf format.
func (bb *BlockBuilder) BuildProto() *corepb.Block {
	n := bb.copy()
	t := bb.t

	pb, err := n.B.ToProto()
	require.NoError(t, err)
	return pb.(*corepb.Block)
}

// BuildBytes builds block in bytes.
func (bb *BlockBuilder) BuildBytes() []byte {
	n := bb.copy()
	t := bb.t

	pb, err := n.B.ToProto()
	require.NoError(t, err)
	data, err := proto.Marshal(pb)
	require.NoError(t, err)
	return data
}

//func (bb *BlockBuilderOld) TransitionDynasty() *BlockBuilderOld {
//	dposState, err := bb.parent.State().DposState().Clone()
//	require.Nil(bb.t, err)
//
//	d := dpos.New()
//	d.SetDynastySize(dynastySize)
//	dynasty, err := d.MakeMintBlockDynasty(bb.Timestamp(), bb.parent)
//	require.NoError(bb.t, err)
//	dpos.SetDynastyState(dposState.DynastyState(), dynasty)
//	require.NoError(bb.t, err)
//	dposRoot, err := dposState.RootBytes()
//	require.NoError(bb.t,err)
//	bb.SetDposRoot(dposRoot)
//
//	return bb
//}
//// Sign generates and sets block's signature.
//func (bb *BlockBuilderOld) SignKey(key signature.PrivateKey) *BlockBuilderOld {
//	require.NotNil(bb.t, bb.pb.Header.Hash)
//
//	signer, err := crypto.NewSignature(defaultSignAlg)
//	require.NoError(bb.t, err)
//	signer.InitSign(key)
//
//	bb.pb.Header.Alg = uint32(signer.Algorithm())
//
//	sig, err := signer.Sign(bb.pb.Header.Hash)
//	require.NoError(bb.t, err)
//	bb.pb.Header.Sign = sig
//
//	return bb
//}

