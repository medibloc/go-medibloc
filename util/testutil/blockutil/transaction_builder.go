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
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/require"
)

//TxBuilder is a structure for transaction builder
type TxBuilder struct {
	t  *testing.T
	bb *BlockBuilder
	tx *core.Transaction
}

func newTxBuilder(bb *BlockBuilder) *TxBuilder {
	n := bb.copy()

	tx := &core.Transaction{}
	tx.SetChainID(testutil.ChainID)
	tx.SetValue(util.Uint128Zero())

	return &TxBuilder{
		t:  bb.t,
		bb: n,
		tx: tx,
	}
}

func (tb *TxBuilder) copy() *TxBuilder {
	var tx *core.Transaction
	var err error
	if tb.tx != nil {
		tx, err = tb.tx.Clone()
		require.NoError(tb.t, err)
	}
	return &TxBuilder{
		t:  tb.t,
		bb: tb.bb,
		tx: tx,
	}
}

/* Setters and Getters */

//Hash sets hash
func (tb *TxBuilder) Hash(hash []byte) *TxBuilder {
	n := tb.copy()
	n.tx.SetHash(hash)
	return n
}

//From sets from
func (tb *TxBuilder) From(addr common.Address) *TxBuilder {
	n := tb.copy()
	n.tx.SetFrom(addr)
	return n
}

//To sets to
func (tb *TxBuilder) To(addr common.Address) *TxBuilder {
	n := tb.copy()
	n.tx.SetTo(addr)
	return n
}

//Value sets value
func (tb *TxBuilder) Value(med float64) *TxBuilder {
	n := tb.copy()
	value := FloatToUint128(n.t, med)
	n.tx.SetValue(value)
	return n
}

//Type sets type
func (tb *TxBuilder) Type(txType string) *TxBuilder {
	n := tb.copy()
	n.tx.SetTxType(txType)
	return n
}

//Payload sets payload
func (tb *TxBuilder) Payload(payload core.TransactionPayload) *TxBuilder {
	n := tb.copy()
	t := tb.t

	b, err := payload.ToBytes()
	require.NoError(t, err)
	n.tx.SetPayload(b)
	return n
}

//Nonce sets nonce
func (tb *TxBuilder) Nonce(nonce uint64) *TxBuilder {
	n := tb.copy()
	n.tx.SetNonce(nonce)
	return n
}

//ChainID sets chainID
func (tb *TxBuilder) ChainID(chainID uint32) *TxBuilder {
	n := tb.copy()
	n.tx.SetChainID(chainID)
	return n
}

//Sign sets sign
func (tb *TxBuilder) Sign(sign []byte) *TxBuilder {
	n := tb.copy()
	n.tx.SetSign(sign)
	return n
}

//PayerSign sets payerSign
func (tb *TxBuilder) PayerSign(sign []byte) *TxBuilder {
	n := tb.copy()
	n.tx.SetPayerSign(sign)
	return n
}

/* Additional Commands */

//CalcHash calculate hash
func (tb *TxBuilder) CalcHash() *TxBuilder {
	n := tb.copy()
	t := tb.t

	hash, err := n.tx.CalcHash()
	require.NoError(t, err)
	n.tx.SetHash(hash)
	return n
}

//SignKey sign by private key
func (tb *TxBuilder) SignKey(key signature.PrivateKey) *TxBuilder {
	n := tb.copy()
	t := tb.t

	signer := signer(t, key)

	sig, err := signer.Sign(n.tx.Hash())
	require.NoError(t, err)
	n.tx.SetSign(sig)
	return n
}

//SignPayerKey sign by payer private key
func (tb *TxBuilder) SignPayerKey(key signature.PrivateKey) *TxBuilder {
	n := tb.copy()
	t := tb.t

	signer := signer(t, key)
	require.NoError(t, n.tx.SignByPayer(signer))
	return n
}

//SignPair set from, seal ,calculate hash and sign with key pair
func (tb *TxBuilder) SignPair(pair *testutil.AddrKeyPair) *TxBuilder {
	n := tb.copy()
	if n.tx.Nonce() == 0 {
		acc, err := n.bb.B.State().GetAccount(pair.Addr)
		require.NoError(n.t, err)
		n = n.Nonce(acc.Nonce + 1)
	}
	return n.From(pair.Addr).CalcHash().SignKey(pair.PrivKey)
}

//RandomTx generate random Tx
func (tb *TxBuilder) RandomTx() *TxBuilder {
	n := tb.copy()
	require.NotEqual(n.t, 0, len(n.bb.KeyPairs), "No key pair added on block builder")

	from := n.bb.KeyPairs[0]
	to := testutil.NewAddrKeyPair(n.t)
	return n.Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from)
}

//RandomTxs generate random transactions
func (tb *TxBuilder) RandomTxs(n int) []*core.Transaction {
	require.NotEqual(tb.t, 0, len(tb.bb.KeyPairs), "No key pair added on block builder")
	txs := make([]*core.Transaction, 0, n)

	from := tb.bb.KeyPairs[0]
	acc, err := tb.bb.B.State().GetAccount(from.Addr)
	require.NoError(tb.t, err)
	nonce := acc.Nonce + 1
	for i := 0; i < n; i++ {
		txs = append(txs, tb.Nonce(nonce+uint64(i)).RandomTx().Build())
	}
	return txs
}

//StakeTx generate stake Tx
func (tb *TxBuilder) StakeTx(pair *testutil.AddrKeyPair, med float64) *TxBuilder {
	n := tb.copy()
	return n.Type(core.TxOpStake).Value(med).SignPair(pair)
}

//Build build transaction
func (tb *TxBuilder) Build() *core.Transaction {
	n := tb.copy()
	return n.tx
}

// BuildCtx build transaction context.
func (tb *TxBuilder) BuildCtx() *core.TxContext {
	n := tb.copy()
	txc, err := core.NewTxContext(n.tx)
	require.NoError(tb.t, err)
	return txc
}

//BuildProto build protobuf transaction
func (tb *TxBuilder) BuildProto() *corepb.Transaction {
	n := tb.copy()
	t := tb.t

	pb, err := n.tx.ToProto()
	require.NoError(t, err)
	return pb.(*corepb.Transaction)
}

//BuildBytes build marshaled transaction
func (tb *TxBuilder) BuildBytes() []byte {
	n := tb.copy()
	t := tb.t

	pb, err := n.tx.ToProto()
	require.NoError(t, err)
	data, err := proto.Marshal(pb)
	require.NoError(t, err)
	return data
}

//Execute excute transaction on block builder
func (tb *TxBuilder) Execute() *BlockBuilder {
	n := tb.copy()
	return n.bb.ExecuteTx(n.tx)
}

//ExecuteErr expect error occurred on executing
func (tb *TxBuilder) ExecuteErr(err error) *BlockBuilder {
	n := tb.copy()
	return n.bb.ExecuteTxErr(n.tx, err)
}

//Add add transaction on block builder
func (tb *TxBuilder) Add() *BlockBuilder {
	n := tb.copy()
	return n.bb.AddTx(n.tx)
}
