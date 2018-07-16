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
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/mitchellh/copystructure"
	"github.com/stretchr/testify/require"
)

type BlockBuilder struct {
	t *testing.T
	B *core.Block

	dynastySize int
	Dynasties   testutil.AddrKeyPairs
	TokenDist   testutil.AddrKeyPairs
	KeyPairs    testutil.AddrKeyPairs
}

func New(t *testing.T, dynastySize int) *BlockBuilder {
	return &BlockBuilder{
		t:           t,
		dynastySize: dynastySize,
	}
}

func (bb *BlockBuilder) AddKeyPairs(keyPairs testutil.AddrKeyPairs) *BlockBuilder {
	n := bb.copy()
	n.KeyPairs = append(n.KeyPairs, keyPairs...)
	return n
}

func (bb *BlockBuilder) copy() *BlockBuilder {
	var b *core.Block
	var err error

	if bb.B != nil {
		b, err = bb.B.Clone()
		require.NoError(bb.t, err)
	}

	dynasties, err := copystructure.Copy(bb.Dynasties)
	require.NoError(bb.t, err)
	tokenDist, err := copystructure.Copy(bb.TokenDist)
	require.NoError(bb.t, err)
	keyPairs, err := copystructure.Copy(bb.KeyPairs)
	require.NoError(bb.t, err)

	return &BlockBuilder{
		t:           bb.t,
		B:           b,
		dynastySize: bb.dynastySize,
		Dynasties:   dynasties.(testutil.AddrKeyPairs),
		TokenDist:   tokenDist.(testutil.AddrKeyPairs),
		KeyPairs:    keyPairs.(testutil.AddrKeyPairs),
	}
}

func (bb *BlockBuilder) Genesis() *BlockBuilder {
	n := bb.copy()
	genesis, dynasties, tokenDist := testutil.NewTestGenesisBlock(bb.t, bb.dynastySize)
	n.B = genesis
	n.Dynasties = dynasties
	n.TokenDist = tokenDist
	n.KeyPairs = append(n.Dynasties, n.TokenDist...)
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
func (bb *BlockBuilder) TransactionRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetTxsRoot(root)
	return n
}

// UsageRoot sets usage root.
func (bb *BlockBuilder) UsageRoot(root []byte) *BlockBuilder {
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
func (bb *BlockBuilder) DposRoot(root []byte) *BlockBuilder {
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
func (bb *BlockBuilder) Coinbase(addr common.Address) *BlockBuilder {
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
func (bb *BlockBuilder) ChainID(chainID uint32) *BlockBuilder {
	n := bb.copy()
	n.B.SetChainID(chainID)
	return n
}

// Alg sets crypto algorithm.
func (bb *BlockBuilder) Alg(alg algorithm.Algorithm) *BlockBuilder {
	n := bb.copy()
	n.B.SetAlg(alg)
	return n
}

// Sign sets signature.
func (bb *BlockBuilder) Sign(sign []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetSign(sign)
	return n
}

// Height sets block height.
func (bb *BlockBuilder) Height(height uint64) *BlockBuilder {
	n := bb.copy()
	n.B.SetHeight(height)
	return n
}

func (bb *BlockBuilder) Sealed(sealed bool) *BlockBuilder {
	n := bb.copy()
	n.B.SetSealed(sealed)
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

func (bb *BlockBuilder) SignPair(pair *testutil.AddrKeyPair) *BlockBuilder {
	n := bb.copy()

	return n.Coinbase(pair.Addr).Seal().CalcHash().SignKey(pair.PrivKey)
}

func (bb *BlockBuilder) SignMiner() *BlockBuilder {
	n := bb.copy()

	proposer, err := n.B.Consensus().FindMintProposer(n.B.Timestamp(), n.B)
	require.NoError(n.t, err)
	pair := n.KeyPairs.FindPair(proposer)
	require.NotNil(n.t, pair)

	return n.SignPair(pair)
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

	require.NoError(n.t, n.B.BeginBatch())
	err := n.B.Execute(tx, defaultTxMap)
	require.NoError(n.t, err)
	err = n.B.AcceptTransaction(tx)
	require.NoError(n.t, err)
	require.NoError(bb.t, n.B.Commit())

	return n
}

func (bb *BlockBuilder) ExecuteTxErr(tx *core.Transaction, expected error) *BlockBuilder {
	n := bb.copy()

	require.NoError(n.t, n.B.BeginBatch())
	err := n.B.Execute(tx, defaultTxMap)
	if err != nil {
		require.Equal(n.t, expected, err)
		return n
	}
	err = n.B.AcceptTransaction(tx)
	require.Equal(n.t, expected, err)
	require.NoError(bb.t, n.B.Commit())

	return n
}

func (bb *BlockBuilder) Seal() *BlockBuilder {
	n := bb.copy()
	t := bb.t
	err := n.B.Seal()
	require.NoError(t, err)
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

func (bb *BlockBuilder) Child() *BlockBuilder {
	n := bb.copy()
	return n.ChildWithTimestamp(time.Unix(bb.B.Timestamp(), 0).Add(dpos.BlockInterval).Unix())
}

func (bb *BlockBuilder) ChildWithTimestamp(ts int64) *BlockBuilder {
	n := bb.copy()
	n.B.SetTransactions(make([]*core.Transaction, 0))
	return n.ParentHash(bb.B.Hash()).Timestamp(ts).Height(bb.B.Height() + 1).Sealed(false).UpdateDynastyState()
}

func (bb *BlockBuilder) UpdateDynastyState() *BlockBuilder {
	n := bb.copy()
	dposState := n.B.State().DposState()

	d := dpos.New()
	d.SetDynastySize(n.dynastySize)
	dynasty, err := d.MakeMintDynasty(n.B.Timestamp(), n.B)
	require.NoError(n.t, err)
	dpos.SetDynastyState(dposState.DynastyState(), dynasty)
	require.NoError(n.t, err)

	return n
}

func (bb *BlockBuilder) Expect() *Expect {
	return NewExpect(bb.t, bb.Build())
}
