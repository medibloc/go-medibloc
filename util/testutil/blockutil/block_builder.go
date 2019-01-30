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
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/stretchr/testify/require"
)

// BlockBuilder is structure for building block
type BlockBuilder struct {
	t *testing.T
	B *core.BlockTestWrap

	dynastySize int
	proposer    common.Address
	Dynasties   keyutil.AddrKeyPairs
	TokenDist   keyutil.AddrKeyPairs
	KeyPairs    keyutil.AddrKeyPairs
}

// New returns new block builder
func New(t *testing.T, dynastySize int) *BlockBuilder {
	return &BlockBuilder{
		t:           t,
		dynastySize: dynastySize,
	}
}

// AddKeyPairs sets key pairs on block builder
func (bb *BlockBuilder) AddKeyPairs(keyPairs keyutil.AddrKeyPairs) *BlockBuilder {
	n := bb.copy()
	n.KeyPairs = append(n.KeyPairs, keyPairs...)
	return n
}

func (bb *BlockBuilder) copy() *BlockBuilder {
	return &BlockBuilder{
		t:           bb.t,
		B:           bb.B,
		dynastySize: bb.dynastySize,
		proposer:    bb.proposer,
		Dynasties:   bb.Dynasties,
		TokenDist:   bb.TokenDist,
		KeyPairs:    bb.KeyPairs,
	}
}

// Clone clones block on block builders (State of block must unprepared)
func (bb *BlockBuilder) Clone() *BlockBuilder {
	n := bb.copy()

	var err error
	require.NotNil(n.t, bb.B)
	n.B, err = bb.B.Clone()
	require.NoError(bb.t, err)
	return n
}

// Genesis create genesis block
func (bb *BlockBuilder) Genesis() *BlockBuilder {
	n := bb.copy()
	genesis, dynasties, tokenDist := NewTestGenesisBlock(bb.t, bb.dynastySize)
	n.B = &core.BlockTestWrap{Block: genesis}
	n.Dynasties = dynasties
	n.TokenDist = tokenDist
	n.KeyPairs = append(n.Dynasties, n.TokenDist...)
	return n
}

// Block sets block
func (bb *BlockBuilder) Block(b *core.Block) *BlockBuilder {
	n := bb.copy()
	proposer, err := b.Proposer()
	if err == core.ErrBlockSignatureNotExist {
		proposer = common.Address{}
	} else {
		require.NoError(n.t, err)
	}

	n.B = &core.BlockTestWrap{Block: b}
	n.proposer = proposer
	return n
}

// Prepare prepare block state
func (bb *BlockBuilder) Prepare() *BlockBuilder {
	n := bb.copy()
	require.NoError(n.t, n.B.Prepare())
	return n
}

// Stake executes stake transactions.
func (bb *BlockBuilder) Stake() *BlockBuilder {
	n := bb.copy()
	for _, pair := range bb.KeyPairs {
		n = n.Tx().StakeTx(pair, 100000).Execute()
	}
	return n
}

// Tx sets tx
func (bb *BlockBuilder) Tx() *TxBuilder {
	return newTxBuilder(bb)
}

// Hash sets hash
func (bb *BlockBuilder) Hash(hash []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetHash(hash)
	return n
}

// ParentHash sets parenthash
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
func (bb *BlockBuilder) Supply(supply string) *BlockBuilder {
	n := bb.copy()
	sp, err := util.NewUint128FromString(supply)
	require.NoError(n.t, err)
	n.B.SetSupply(sp)
	return n
}

// AccountRoot sets account root.
func (bb *BlockBuilder) AccountRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetAccStateRoot(root)
	return n
}

// TxRoot sets data state root.
func (bb *BlockBuilder) TxRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetTxStateRoot(root)
	return n
}

// DposRoot sets dpos root.
func (bb *BlockBuilder) DposRoot(root []byte) *BlockBuilder {
	n := bb.copy()
	n.B.SetDposRoot(root)
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
	n.B.State().SetTimestamp(ts)
	return n
}

// ChainID sets chain ID.
func (bb *BlockBuilder) ChainID(chainID uint32) *BlockBuilder {
	n := bb.copy()
	n.B.SetChainID(chainID)
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

// Sealed sets sealed
func (bb *BlockBuilder) Sealed(sealed bool) *BlockBuilder {
	n := bb.copy()
	n.B.SetSealed(sealed)
	return n
}

// CalcHash calculate hash
func (bb *BlockBuilder) CalcHash() *BlockBuilder {
	n := bb.copy()
	hash, err := n.B.CalcHash()
	require.NoError(bb.t, err)
	n.B.SetHash(hash)
	return n
}

// SignKey signs by private key
func (bb *BlockBuilder) SignKey(key signature.PrivateKey) *BlockBuilder {
	n := bb.copy()
	t := bb.t

	signer := signer(t, key)

	sig, err := signer.Sign(n.B.Hash())
	require.NoError(t, err)
	n.B.SetSign(sig)
	return n
}

// SignPair set coinbase, seal, calculate hash and sign with key pair
func (bb *BlockBuilder) SignPair(pair *keyutil.AddrKeyPair) *BlockBuilder {
	n := bb.copy()

	return n.Coinbase(pair.Addr).PayReward().Flush().Seal().CalcHash().SignKey(pair.PrivKey)
}

// SignProposer find proposer and sign with key pair
func (bb *BlockBuilder) SignProposer() *BlockBuilder {
	n := bb.copy()

	return n.SignPair(n.FindProposer())
}

// FindProposer finds proposer.
func (bb *BlockBuilder) FindProposer() *keyutil.AddrKeyPair {
	require.NotNil(bb.t, bb.proposer)
	pair := bb.KeyPairs.FindPair(bb.proposer)
	require.NotNil(bb.t, pair)

	return pair
}

// AddTx add transaction
func (bb *BlockBuilder) AddTx(tx *core.Transaction) *BlockBuilder {
	n := bb.copy()
	txs := n.B.Transactions()
	txs = append(txs, tx)
	require.NoError(n.t, n.B.SetTransactions(txs))
	return n
}

// ExecuteTx execute transaction
func (bb *BlockBuilder) ExecuteTx(tx *core.Transaction) *BlockBuilder {
	n := bb.copy()
	receipt, err := n.B.ExecuteTransaction(tx)
	require.NoError(n.t, err)
	require.Nil(n.t, receipt.Error())
	require.True(n.t, receipt.Executed())

	tx.SetReceipt(receipt)
	require.NoError(n.t, n.B.AcceptTransaction(tx))
	n.B.AppendTransaction(tx)
	return n
}

// ExecuteTxErr expect error occurred on executing
func (bb *BlockBuilder) ExecuteTxErr(tx *core.Transaction, expected error) *BlockBuilder {
	n := bb.copy()
	receipt, err := n.B.ExecuteTransaction(tx)
	if err != nil {
		require.Equal(n.t, expected, err)
		return n
	}
	require.Equal(n.t, expected.Error(), string(receipt.Error()))
	require.False(n.t, receipt.Executed())

	tx.SetReceipt(receipt)
	require.NoError(n.t, n.B.AcceptTransaction(tx))
	n.B.AppendTransaction(tx)
	return n
}

// PayReward pay reward and update reward and supply
func (bb *BlockBuilder) PayReward() *BlockBuilder {
	n := bb.copy()

	require.NoError(n.t, n.B.PayReward(n.B.Supply()))

	return n
}

// TODO remove prepare, flush calls
// Flush saves state to storage
func (bb *BlockBuilder) Flush() *BlockBuilder {
	n := bb.copy()
	require.NoError(n.t, n.B.Flush())
	return n
}

// Seal set root hash on header
func (bb *BlockBuilder) Seal() *BlockBuilder {
	n := bb.copy()
	require.NoError(n.t, n.B.Seal())
	return n
}

// Build builds block.
func (bb *BlockBuilder) Build() *core.Block {
	n := bb.copy()
	return n.B.Block
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

// Child create child block
func (bb *BlockBuilder) Child() *BlockBuilder {
	n := bb.copy()
	return n.ChildWithTimestamp(time.Unix(bb.B.Timestamp(), 0).Add(dpos.BlockInterval).Unix())
}

// ChildWithTimestamp create child block on specific timestamp
func (bb *BlockBuilder) ChildWithTimestamp(ts int64) *BlockBuilder {
	n := bb.copy()
	var err error
	parent := bb.B
	n.B, err = parent.InitChild(common.Address{})
	require.NoError(n.t, err)
	return n.Timestamp(ts).Prepare().UpdateDynastyState(parent.Block)
}

// ChildNextDynasty create first child block of next dynasty
func (bb *BlockBuilder) ChildNextDynasty() *BlockBuilder {
	n := bb.copy()

	d := dpos.New(n.dynastySize)
	curDynastyIndex := int(time.Duration(n.B.Timestamp()) * time.Second / d.DynastyInterval())
	nextTs := int64(time.Duration(curDynastyIndex+1) * d.DynastyInterval() / time.Second)

	return n.ChildWithTimestamp(nextTs)
}

// UpdateDynastyState update dynasty state
func (bb *BlockBuilder) UpdateDynastyState(parent *core.Block) *BlockBuilder {
	n := bb.copy()
	d := dpos.New(n.dynastySize)

	mintProposer, err := d.FindMintProposer(n.B.Timestamp(), parent)
	require.NoError(n.t, err)
	n.proposer = mintProposer

	require.NoError(n.t, n.B.SetMintDynasty(parent, d))

	return n
}

// Expect return expect
func (bb *BlockBuilder) Expect() *Expect {
	return NewExpect(bb.t, bb.Build())
}
