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
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

//TxBuilder is a structure for transaction builder
type TxBuilder struct {
	t  *testing.T
	bb *BlockBuilder
	tx *core.Transaction
}

func newTxBuilder(bb *BlockBuilder) *TxBuilder {
	n := bb.copy()
	tx := defaultTransaction()
	tx.SetTimestamp(bb.B.Timestamp())

	return &TxBuilder{
		t:  bb.t,
		bb: n,
		tx: tx,
	}
}

func defaultTransaction() *core.Transaction {
	tx := &core.Transaction{}
	tx.SetChainID(testutil.ChainID)
	tx.SetValue(util.Uint128Zero())
	tx.SetData(&corepb.Data{})
	tx.SetAlg(defaultSignAlg)
	return tx
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
func (tb *TxBuilder) Value(value uint64) *TxBuilder {
	n := tb.copy()
	n.tx.SetValue(util.NewUint128FromUint(value))
	return n
}

//Timestamp sets timestamp
func (tb *TxBuilder) Timestamp(ts int64) *TxBuilder {
	n := tb.copy()
	n.tx.SetTimestamp(ts)
	return n
}

//Type sets type
func (tb *TxBuilder) Type(txType string) *TxBuilder {
	n := tb.copy()
	n.tx.SetType(txType)
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

//Alg sets signature algorithm
func (tb *TxBuilder) Alg(alg algorithm.Algorithm) *TxBuilder {
	n := tb.copy()
	n.tx.SetAlg(alg)
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

	n.tx.SetAlg(signer.Algorithm())

	sig, err := signer.Sign(n.tx.Hash())
	require.NoError(t, err)
	n.tx.SetSign(sig)
	return n
}

//SignPayerKey sign by payer private key
func (tb *TxBuilder) SignPayerKey(key signature.PrivateKey) *TxBuilder {
	n := tb.copy()
	t := tb.t

	hasher := sha3.New256()
	hasher.Write(n.tx.Hash())
	hasher.Write(n.tx.Sign())
	hash := hasher.Sum(nil)

	signer := signer(t, key)
	sig, err := signer.Sign(hash)
	require.NoError(t, err)

	n.tx.SetPayerSign(sig)
	return n
}

//SignPair set from, seal ,calculate hash and sign with key pair
func (tb *TxBuilder) SignPair(pair *testutil.AddrKeyPair) *TxBuilder {
	n := tb.copy()
	if n.tx.Nonce() == 0 {
		acc, err := n.bb.B.State().GetAccount(pair.Addr)
		require.NoError(n.t, err)
		n = n.Nonce(acc.Nonce() + 1)
	}
	return n.From(pair.Addr).CalcHash().SignKey(pair.PrivKey)
}

//RandomTx generate random Tx
func (tb *TxBuilder) RandomTx() *TxBuilder {
	n := tb.copy()
	from := testutil.NewAddrKeyPair(n.t)
	to := testutil.NewAddrKeyPair(n.t)
	return n.Type(core.TxOpTransfer).Value(10).To(to.Addr).SignPair(from)
}

//Build build transaction
func (tb *TxBuilder) Build() *core.Transaction {
	n := tb.copy()
	return n.tx
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
