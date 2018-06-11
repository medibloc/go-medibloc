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
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package core

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/crypto/sha3"
)

// Transaction struct represents transaction
type Transaction struct {
	hash      []byte
	from      common.Address
	to        common.Address
	value     *util.Uint128
	timestamp int64
	data      *corepb.Data
	nonce     uint64
	chainID   uint32
	alg       algorithm.Algorithm
	sign      []byte
	payerSign []byte
}

// Transactions is just multiple txs
type Transactions []*Transaction

// ToProto converts Transaction to corepb.Transaction
func (tx *Transaction) ToProto() (proto.Message, error) {
	value, err := tx.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	return &corepb.Transaction{
		Hash:      tx.hash,
		From:      tx.from.Bytes(),
		To:        tx.to.Bytes(),
		Value:     value,
		Timestamp: tx.timestamp,
		Data:      tx.data,
		Nonce:     tx.nonce,
		ChainId:   tx.chainID,
		Alg:       uint32(tx.alg),
		Sign:      tx.sign,
		PayerSign: tx.payerSign,
	}, nil
}

// FromProto converts corepb.Transaction to Transaction
func (tx *Transaction) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Transaction); ok {
		tx.hash = msg.Hash
		tx.from = common.BytesToAddress(msg.From)
		tx.to = common.BytesToAddress(msg.To)

		value, err := util.NewUint128FromFixedSizeByteSlice(msg.Value)
		if err != nil {
			return err
		}
		tx.value = value
		tx.timestamp = msg.Timestamp
		tx.data = msg.Data
		tx.nonce = msg.Nonce
		tx.chainID = msg.ChainId
		alg := algorithm.Algorithm(msg.Alg)
		err = crypto.CheckAlgorithm(alg)
		if err != nil {
			return err
		}
		tx.alg = alg
		tx.sign = msg.Sign
		tx.payerSign = msg.PayerSign

		return nil
	}

	return ErrCannotConvertTransaction
}

// BuildTransaction generates a Transaction instance with entire fields
func BuildTransaction(
	chainID uint32,
	from, to common.Address,
	value *util.Uint128,
	nonce uint64,
	timestamp int64,
	data *corepb.Data,
	hash []byte,
	alg uint32,
	sign []byte,
	payerSign []byte) (*Transaction, error) {
	return &Transaction{
		from:      from,
		to:        to,
		value:     value,
		timestamp: timestamp,
		data:      data,
		nonce:     nonce,
		chainID:   chainID,
		hash:      hash,
		alg:       algorithm.Algorithm(alg),
		sign:      sign,
		payerSign: payerSign,
	}, nil
}

// NewTransaction generates a Transaction instance
func NewTransaction(
	chainID uint32,
	from, to common.Address,
	value *util.Uint128,
	nonce uint64,
	payloadType string,
	payload []byte) (*Transaction, error) {
	return NewTransactionWithSign(chainID, from, to, value, nonce, payloadType, payload, []byte{}, []byte{}, []byte{})
}

// NewTransactionWithSign generates a Transaction instance with sign
func NewTransactionWithSign(
	chainID uint32,
	from, to common.Address,
	value *util.Uint128,
	nonce uint64,
	payloadType string,
	payload []byte,
	hash []byte,
	sign []byte,
	payerSign []byte) (*Transaction, error) {

	return &Transaction{
		from:      from,
		to:        to,
		value:     value,
		timestamp: time.Now().Unix(),
		data:      &corepb.Data{Type: payloadType, Payload: payload},
		nonce:     nonce,
		chainID:   chainID,
		hash:      hash,
		sign:      sign,
		payerSign: payerSign,
	}, nil
}

func (tx *Transaction) calcHash() ([]byte, error) {
	hasher := sha3.New256()

	value, err := tx.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	hasher.Write(tx.from.Bytes())
	hasher.Write(tx.to.Bytes())
	hasher.Write(value)
	hasher.Write(byteutils.FromInt64(tx.timestamp))
	hasher.Write([]byte(tx.data.Type))
	hasher.Write(tx.data.Payload)
	hasher.Write(byteutils.FromUint64(tx.nonce))
	hasher.Write(byteutils.FromUint32(tx.chainID))
	hasher.Write(byteutils.FromUint32(uint32(tx.alg)))

	hash := hasher.Sum(nil)
	return hash, nil
}

// SignThis signs tx with given signature interface
func (tx *Transaction) SignThis(signer signature.Signature) error {
	tx.alg = signer.Algorithm()
	hash, err := tx.calcHash()
	if err != nil {
		return err
	}

	sig, err := signer.Sign(hash)
	if err != nil {
		return err
	}
	tx.hash = hash
	tx.sign = sig
	return nil
}

func (tx *Transaction) getPayerSignTarget() []byte {
	return append(tx.hash, tx.sign...)
}

// SignByPayer puts payer's sign in tx
func (tx *Transaction) SignByPayer(signer signature.Signature) error {
	target := tx.getPayerSignTarget()

	sig, err := signer.Sign(target)
	if err != nil {
		return err
	}
	tx.payerSign = sig
	return nil
}

// VerifyIntegrity returns transaction verify result, including Hash and Signature.
func (tx *Transaction) VerifyIntegrity(chainID uint32) error {
	// check ChainID.
	if tx.chainID != chainID {
		return ErrInvalidChainID
	}

	// check Hash.
	wantedHash, err := tx.calcHash()
	if err != nil {
		return err
	}
	if !byteutils.Equal(wantedHash, tx.hash) {
		return ErrInvalidTransactionHash
	}

	// check Signature.
	return tx.verifySign()
}

func (tx *Transaction) verifySign() error {
	signer, err := tx.recoverSigner()
	if err != nil {
		return err
	}
	if !tx.from.Equals(signer) {
		return ErrInvalidTransactionSigner
	}
	return nil
}

func (tx *Transaction) recoverSigner() (common.Address, error) {
	signature, err := crypto.NewSignature(tx.alg)
	if err != nil {
		return common.Address{}, err
	}

	pubKey, err := signature.RecoverPublic(tx.hash, tx.sign)
	if err != nil {
		return common.Address{}, err
	}

	return common.PublicKeyToAddress(pubKey)
}

// From returns from
func (tx *Transaction) From() common.Address {
	return tx.from
}

// To returns to
func (tx *Transaction) To() common.Address {
	return tx.to
}

// Value returns value
func (tx *Transaction) Value() *util.Uint128 {
	return tx.value
}

// Timestamp returns tx timestamp
func (tx *Transaction) Timestamp() int64 {
	return tx.timestamp
}

// SetTimestamp sets tx timestamp
func (tx *Transaction) SetTimestamp(t int64) {
	tx.timestamp = t
}

// Type returns tx type
func (tx *Transaction) Type() string {
	return tx.data.Type
}

// Data returns tx payload
func (tx *Transaction) Data() []byte {
	return tx.data.Payload
}

// Nonce returns tx nonce
func (tx *Transaction) Nonce() uint64 {
	return tx.nonce
}

// Hash returns hash
func (tx *Transaction) Hash() []byte {
	return tx.hash
}

// Signature returns tx signature
func (tx *Transaction) Signature() []byte {
	return tx.sign
}

// PayerSignature returns tx payer signature
func (tx *Transaction) PayerSignature() []byte {
	return tx.payerSign
}

// String returns string representation of tx
func (tx *Transaction) String() string {
	return fmt.Sprintf(`{chainID:%v, hash:%v, from:%v, to:%v, value:%v, type:%v, alg:%v, nonce:%v'}`,
		tx.chainID,
		tx.hash,
		tx.from,
		tx.to,
		tx.value.String(),
		tx.Type(),
		tx.alg,
		tx.nonce,
	)
}

// ExecuteOnState executes tx on block state and change the state if valid
func (tx *Transaction) ExecuteOnState(bs *BlockState) error {
	if err := bs.checkNonce(tx); err != nil {
		return err
	}

	switch tx.Type() {
	case TxOperationAddRecord:
		return tx.addRecord(bs)
	case TxOperationVest:
		return tx.vest(bs)
	case TxOperationWithdrawVesting:
		return tx.withdrawVesting(bs)
	case TxOperationBecomeCandidate:
		return tx.becomeCandidate(bs)
	case TxOperationQuitCandidacy:
		return tx.quitCandidacy(bs)
	case TxOperationVote:
		return tx.vote(bs)
	default:
		return tx.transfer(bs)
	}
}

func (tx *Transaction) transfer(bs *BlockState) error {
	if tx.value.Cmp(util.Uint128Zero()) == 0 {
		return ErrVoidTransaction
	}

	var err error
	if err = bs.SubBalance(tx.from, tx.value); err != nil {
		return err
	}
	return bs.AddBalance(tx.to, tx.value)
}

func (tx *Transaction) addRecord(bs *BlockState) error {
	payload, err := BytesToAddRecordPayload(tx.Data())
	if err != nil {
		return err
	}
	return bs.AddRecord(tx, payload.Hash, tx.from)
}

func (tx *Transaction) vest(bs *BlockState) error {
	return bs.Vest(tx.from, tx.value)
}

func (tx *Transaction) withdrawVesting(bs *BlockState) error {
	return bs.WithdrawVesting(tx.from, tx.value, tx.timestamp)
}

func (tx *Transaction) becomeCandidate(bs *BlockState) error {
	return bs.AddCandidate(tx.from, tx.value)
}

func (tx *Transaction) quitCandidacy(bs *BlockState) error {
	return bs.QuitCandidacy(tx.from)
}

func (tx *Transaction) vote(bs *BlockState) error {
	return bs.Vote(tx.from, tx.to)
}
