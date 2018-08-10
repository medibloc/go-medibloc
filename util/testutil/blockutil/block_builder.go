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
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/require"
)

//BlockBuilder is structure for building block
type BlockBuilder struct {
	t *testing.T
	B *core.Block

	dynastySize int
	proposer    common.Address
	Dynasties   testutil.AddrKeyPairs
	TokenDist   testutil.AddrKeyPairs
	KeyPairs    testutil.AddrKeyPairs
}

//New returns new block builder
func New(t *testing.T, dynastySize int) *BlockBuilder {
	return &BlockBuilder{
		t:           t,
		dynastySize: dynastySize,
	}
}

//AddKeyPairs sets key pairs on block builder
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

	//dynasties, err := copystructure.Copy(bb.Dynasties)
	//require.NoError(bb.t, err)
	//tokenDist, err := copystructure.Copy(bb.TokenDist)
	//require.NoError(bb.t, err)
	//keyPairs, err := copystructure.Copy(bb.KeyPairs)
	//require.NoError(bb.t, err)

	return &BlockBuilder{
		t:           bb.t,
		B:           b,
		dynastySize: bb.dynastySize,
		proposer:    bb.proposer,
		Dynasties:   bb.Dynasties,
		TokenDist:   bb.TokenDist,
		KeyPairs:    bb.KeyPairs,
	}
}

//Genesis create genesis block
func (bb *BlockBuilder) Genesis() *BlockBuilder {
	n := bb.copy()
	genesis, dynasties, tokenDist := testutil.NewTestGenesisBlock(bb.t, bb.dynastySize)
	n.B = genesis
	n.Dynasties = dynasties
	n.TokenDist = tokenDist
	n.KeyPairs = append(n.Dynasties, n.TokenDist...)
	return n
}

//Block sets block
func (bb *BlockBuilder) Block(block *core.Block) *BlockBuilder {
	n := bb.copy()
	proposer, _ := block.Proposer()
	//require.NoError(n.t, err)

	n.B = block
	n.proposer = proposer
	return n
}

//Tx sets tx
func (bb *BlockBuilder) Tx() *TxBuilder {
	return newTxBuilder(bb)
}

//Hash sets hash
func (bb *BlockBuilder) Hash(hash []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetHash(hash)
	return n
}

//ParentHash sets parenthash
func (bb *BlockBuilder) ParentHash(hash []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetParentHash(hash)
	return n
}

// Reward sets reward.
func (bb *BlockBuilder) Reward(reward uint64) *BlockBuilder {
	n := bb.copy()
	n.B.SetReward(util.NewUint128FromUint(reward))
	return n
}

// Supply sets supply.
func (bb *BlockBuilder) Supply(supply uint64) *BlockBuilder {
	n := bb.copy()
	n.B.SetSupply(util.NewUint128FromUint(supply))
	return n
}

// AccountRoot sets account root.
func (bb *BlockBuilder) AccountRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetAccStateRoot(root)
	return n
}

// DataRoot sets data state root.
func (bb *BlockBuilder) DataRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetDataStateRoot(root)
	return n
}

// UsageRoot sets usage root.
func (bb *BlockBuilder) UsageRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetUsageRoot(root)
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

//Sealed sets sealed
func (bb *BlockBuilder) Sealed(sealed bool) *BlockBuilder {
	n := bb.copy()
	n.B.SetSealed(sealed)
	return n
}

//CalcHash calculate hash
func (bb *BlockBuilder) CalcHash() *BlockBuilder {
	n := bb.copy()
	hash := core.HashBlockData(n.B.GetBlockData())
	n.B.SetHash(hash)
	return n
}

//SignKey signs by private key
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

//SignPair set coinbase, seal, calculate hash and sign with key pair
func (bb *BlockBuilder) SignPair(pair *testutil.AddrKeyPair) *BlockBuilder {
	n := bb.copy()

	return n.Coinbase(pair.Addr).PayReward().Seal().CalcHash().SignKey(pair.PrivKey)
}

//SignMiner find proposer and sign with key pair
func (bb *BlockBuilder) SignMiner() *BlockBuilder {
	n := bb.copy()

	return n.SignPair(n.FindMiner())
}

//FindMiner finds proposer.
func (bb *BlockBuilder) FindMiner() *testutil.AddrKeyPair {
	require.NotNil(bb.t, bb.proposer)
	pair := bb.KeyPairs.FindPair(bb.proposer)
	require.NotNil(bb.t, pair)

	return pair
}

//AddTx add transaction
func (bb *BlockBuilder) AddTx(tx *core.Transaction) *BlockBuilder {
	n := bb.copy()
	txs := n.B.Transactions()
	txs = append(txs, tx)
	n.B.SetTransactions(txs)
	return n
}

//ExecuteTx execute transaction
func (bb *BlockBuilder) ExecuteTx(tx *core.Transaction) *BlockBuilder {
	n := bb.copy()

	require.NoError(n.t, n.B.BeginBatch())
	require.NoError(n.t, n.B.ExecuteTransaction(tx, defaultTxMap))
	require.NoError(n.t, n.B.AcceptTransaction(tx))
	require.NoError(bb.t, n.B.Commit())

	return n
}

//ExecuteTxErr expect error occurred on executing
func (bb *BlockBuilder) ExecuteTxErr(tx *core.Transaction, expected error) *BlockBuilder {
	n := bb.copy()

	require.NoError(n.t, n.B.BeginBatch())
	err := n.B.ExecuteTransaction(tx, defaultTxMap)
	if err != nil {
		require.Equal(n.t, expected, err)
		return n
	}
	err = n.B.AcceptTransaction(tx)
	require.Equal(n.t, expected, err)
	require.NoError(bb.t, n.B.Commit())

	return n
}

//PayReward pay reward and update reward and supply
func (bb *BlockBuilder) PayReward() *BlockBuilder {
	n := bb.copy()

	require.NoError(n.t, n.B.BeginBatch())
	require.NoError(n.t, n.B.PayReward(n.B.Coinbase(), n.B.Supply()))
	require.NoError(n.t, n.B.Commit())

	return n
}

//Seal set root hash on header
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

//Child create child block
func (bb *BlockBuilder) Child() *BlockBuilder {
	n := bb.copy()
	return n.ChildWithTimestamp(time.Unix(bb.B.Timestamp(), 0).Add(dpos.BlockInterval).Unix())
}

//ChildWithTimestamp create child block on specific timestamp
func (bb *BlockBuilder) ChildWithTimestamp(ts int64) *BlockBuilder {
	n := bb.copy()
	parent := bb.B
	n.B.SetTransactions(make([]*core.Transaction, 0))
	return n.ParentHash(bb.B.Hash()).Timestamp(ts).Height(bb.B.Height() + 1).Sealed(false).UpdateDynastyState(parent).ExecuteReservedTasks()
}

//ChildNextDynasty create first child block of next dynasty
func (bb *BlockBuilder) ChildNextDynasty() *BlockBuilder {
	n := bb.copy()

	d := dpos.New(n.dynastySize)
	curDynastyIndex := int(time.Duration(n.B.Timestamp()) * time.Second / d.DynastyInterval())
	nextTs := int64(time.Duration(curDynastyIndex+1) * d.DynastyInterval() / time.Second)

	return n.ChildWithTimestamp(nextTs)
}

//UpdateDynastyState update dynasty state
func (bb *BlockBuilder) UpdateDynastyState(parent *core.Block) *BlockBuilder {
	n := bb.copy()
	d := dpos.New(n.dynastySize)

	mintProposer, err := d.FindMintProposer(n.B.Timestamp(), parent)
	require.NoError(n.t, err)
	n.proposer = mintProposer

	require.NoError(n.t, n.B.BeginBatch())
	require.NoError(n.t, n.B.SetMintDposState(parent))
	require.NoError(n.t, n.B.Commit())

	return n
}

//Expect return expect
func (bb *BlockBuilder) Expect() *Expect {
	return NewExpect(bb.t, bb.Build())
}

//ExecuteReservedTasks execute reserved tasks
func (bb *BlockBuilder) ExecuteReservedTasks() *BlockBuilder {
	n := bb.copy()

	require.NoError(n.t, n.B.BeginBatch())
	require.NoError(n.t, n.B.ExecuteReservedTasks())
	require.NoError(bb.t, n.B.Commit())

	return n
}
