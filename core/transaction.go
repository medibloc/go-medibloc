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

var TxBaseBandwidth = util.NewUint128FromUint(100)

// Transaction struct represents transaction
type Transaction struct {
	hash      []byte
	txType    string
	from      common.Address
	to        common.Address
	value     *util.Uint128
	timestamp int64
	nonce     uint64
	chainID   uint32
	payload   []byte
	alg       algorithm.Algorithm
	sign      []byte
	payerSign []byte
}

// ToProto converts Transaction to corepb.Transaction
func (t *Transaction) ToProto() (proto.Message, error) {
	value, err := t.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	return &corepb.Transaction{
		Hash:      t.hash,
		TxType:    t.txType,
		From:      t.from.Bytes(),
		To:        t.to.Bytes(),
		Value:     value,
		Timestamp: t.timestamp,
		Nonce:     t.nonce,
		ChainId:   t.chainID,
		Payload:   t.payload,
		Alg:       uint32(t.alg),
		Sign:      t.sign,
		PayerSign: t.payerSign,
	}, nil
}

// FromProto converts corepb.Transaction to Transaction
func (t *Transaction) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Transaction); ok {
		value, err := util.NewUint128FromFixedSizeByteSlice(msg.Value)
		if err != nil {
			return err
		}

		t.hash = msg.Hash
		t.txType = msg.TxType
		t.from = common.BytesToAddress(msg.From)
		t.to = common.BytesToAddress(msg.To)
		t.value = value
		t.timestamp = msg.Timestamp
		t.nonce = msg.Nonce
		t.chainID = msg.ChainId
		t.payload = msg.Payload
		t.alg = algorithm.Algorithm(msg.Alg)
		t.sign = msg.Sign
		t.payerSign = msg.PayerSign

		return nil
	}

	return ErrCannotConvertTransaction
}

//Hash returns hash
func (t *Transaction) Hash() []byte {
	return t.hash
}

//SetHash sets hash
func (t *Transaction) SetHash(hash []byte) {
	t.hash = hash
}

//TxType returns type
func (t *Transaction) TxType() string {
	return t.txType
}

//SetTxType sets type
func (t *Transaction) SetTxType(txType string) {
	t.txType = txType
}

//From returns from
func (t *Transaction) From() common.Address {
	return t.from
}

//SetFrom sets from
func (t *Transaction) SetFrom(from common.Address) {
	t.from = from
}

//To returns to
func (t *Transaction) To() common.Address {
	return t.to
}

//SetTo sets to
func (t *Transaction) SetTo(to common.Address) {
	t.to = to
}

//Value returns value
func (t *Transaction) Value() *util.Uint128 {
	return t.value
}

//SetValue set value
func (t *Transaction) SetValue(value *util.Uint128) {
	t.value = value
}

//Timestamp returns timestamp
func (t *Transaction) Timestamp() int64 {
	return t.timestamp
}

//SetTimestamp set timestamp
func (t *Transaction) SetTimestamp(timestamp int64) {
	t.timestamp = timestamp
}

//Payload returns paylaod
func (t *Transaction) Payload() []byte {
	return t.payload
}

//SetPayload set payload
func (t *Transaction) SetPayload(payload []byte) {
	t.payload = payload
}

//Nonce returns nounce
func (t *Transaction) Nonce() uint64 {
	return t.nonce
}

//SetNonce set nonce
func (t *Transaction) SetNonce(nonce uint64) {
	t.nonce = nonce
}

//ChainID returns chainID
func (t *Transaction) ChainID() uint32 {
	return t.chainID
}

//SetChainID set chainID
func (t *Transaction) SetChainID(chainID uint32) {
	t.chainID = chainID
}

//Alg returns signing algorithm
func (t *Transaction) Alg() algorithm.Algorithm {
	return t.alg
}

//SetAlg set signing algorithm
func (t *Transaction) SetAlg(alg algorithm.Algorithm) {
	t.alg = alg
}

//Sign returns sign
func (t *Transaction) Sign() []byte {
	return t.sign
}

//SetSign set sign
func (t *Transaction) SetSign(sign []byte) {
	t.sign = sign
}

//PayerSign return payerSign
func (t *Transaction) PayerSign() []byte {
	return t.payerSign
}

//SetPayerSign set payerSign
func (t *Transaction) SetPayerSign(payerSign []byte) {
	t.payerSign = payerSign
}

//IsRelatedToAddress return whether the transaction is related to the address
func (t *Transaction) IsRelatedToAddress(address common.Address) bool {
	if t.from == address || t.to == address {
		return true
	}
	return false
}

// CalcHash calculates transaction's hash.
func (t *Transaction) CalcHash() ([]byte, error) {
	hasher := sha3.New256()

	value, err := t.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	txHashTarget := &corepb.TransactionHashTarget{
		TxType:    t.txType,
		From:      t.from.Bytes(),
		To:        t.to.Bytes(),
		Value:     value,
		Timestamp: t.timestamp,
		Nonce:     t.nonce,
		ChainId:   t.chainID,
		Payload:   t.payload,
	}
	txHashTargetBytes, err := proto.Marshal(txHashTarget)
	if err != nil {
		return nil, err
	}
	hasher.Write(txHashTargetBytes)

	hash := hasher.Sum(nil)
	return hash, nil
}

// SignThis signs tx with given signature interface
func (t *Transaction) SignThis(signer signature.Signature) error {
	t.alg = signer.Algorithm()
	hash, err := t.CalcHash()
	if err != nil {
		return err
	}

	sig, err := signer.Sign(hash)
	if err != nil {
		return err
	}
	t.hash = hash
	t.sign = sig
	return nil
}

func (t *Transaction) getPayerSignTarget() []byte {
	hasher := sha3.New256()

	hasher.Write(t.hash)
	hasher.Write(t.sign)

	hash := hasher.Sum(nil)
	return hash
}

func (t *Transaction) recoverPayer() (common.Address, error) {
	if t.payerSign == nil || len(t.payerSign) == 0 {
		return common.Address{}, ErrPayerSignatureNotExist
	}
	msg := t.getPayerSignTarget()

	sig, err := crypto.NewSignature(t.alg)
	if err != nil {
		return common.Address{}, err
	}

	pubKey, err := sig.RecoverPublic(msg, t.payerSign)
	if err != nil {
		return common.Address{}, err
	}

	return common.PublicKeyToAddress(pubKey)
}

// SignByPayer puts payer's sign in tx
func (t *Transaction) SignByPayer(signer signature.Signature) error {
	target := t.getPayerSignTarget()

	sig, err := signer.Sign(target)
	if err != nil {
		return err
	}
	t.payerSign = sig
	return nil
}

// VerifyIntegrity returns transaction verify result, including Hash and Signature.
func (t *Transaction) VerifyIntegrity(chainID uint32) error {
	// check ChainID.
	if t.chainID != chainID {
		return ErrInvalidChainID
	}

	// check Hash.
	wantedHash, err := t.CalcHash()
	if err != nil {
		return err
	}
	if !byteutils.Equal(wantedHash, t.hash) {
		return ErrInvalidTransactionHash
	}

	// check Signature.
	return t.verifySign()
}

func (t *Transaction) verifySign() error {
	if err := crypto.CheckAlgorithm(t.alg); err != nil {
		return err
	}
	signer, err := t.recoverSigner()
	if err != nil {
		return err
	}
	if !t.from.Equals(signer) {
		return ErrInvalidTransactionSigner
	}
	return nil
}

func (t *Transaction) recoverSigner() (common.Address, error) {
	sig, err := crypto.NewSignature(t.alg)
	if err != nil {
		return common.Address{}, err
	}

	pubKey, err := sig.RecoverPublic(t.hash, t.sign)
	if err != nil {
		return common.Address{}, err
	}

	return common.PublicKeyToAddress(pubKey)
}

// String returns string representation of tx
func (t *Transaction) String() string {
	return fmt.Sprintf(`{chainID:%v, hash:%v, from:%v, to:%v, value:%v, type:%v, alg:%v, nonce:%v, timestamp:%v}`,
		t.chainID,
		byteutils.Bytes2Hex(t.hash),
		t.from,
		t.to,
		t.value.String(),
		t.TxType(),
		t.alg,
		t.nonce,
		t.timestamp,
	)
}

//Clone clone transaction
func (t *Transaction) Clone() (*Transaction, error) {
	protoTx, err := t.ToProto()
	if err != nil {
		return nil, err
	}
	newTx := new(Transaction)
	err = newTx.FromProto(protoTx)
	if err != nil {
		return nil, err
	}
	return newTx, nil
}

//TransferTx is a structure for sending MED
type TransferTx struct {
	from    common.Address
	to      common.Address
	value   *util.Uint128
	payload *DefaultPayload
}

//NewTransferTx returns TransferTx
func NewTransferTx(tx *Transaction) (ExecutableTx, error) {
	if tx.value.Cmp(util.Uint128Zero()) == 0 {
		return nil, ErrVoidTransaction
	}
	payload := new(DefaultPayload)
	if err := BytesToTransactionPayload(tx.payload, payload); err != nil {
		return nil, err
	}

	return &TransferTx{
		from:    tx.From(),
		to:      tx.To(),
		value:   tx.Value(),
		payload: payload,
	}, nil
}

//Execute TransferTx
func (tx *TransferTx) Execute(b *Block) error {
	as := b.state.AccState()

	if err := as.SubBalance(tx.from, tx.value); err != nil {
		return err
	}
	return as.AddBalance(tx.to, tx.value)
}

//Bandwidth returns bandwidth.
func (tx *TransferTx) Bandwidth() (*util.Uint128, error) {
	return TxBaseBandwidth, nil
}

//AddRecordTx is a structure for adding record
type AddRecordTx struct {
	owner      common.Address
	timestamp  int64
	recordHash []byte
}

//NewAddRecordTx returns AddRecordTx
func NewAddRecordTx(tx *Transaction) (ExecutableTx, error) {
	payload := new(AddRecordPayload)
	if err := BytesToTransactionPayload(tx.payload, payload); err != nil {
		return nil, err
	}

	return &AddRecordTx{
		owner:      tx.From(),
		timestamp:  tx.Timestamp(),
		recordHash: payload.RecordHash,
	}, nil
}

//Execute AddRecordTx
func (tx *AddRecordTx) Execute(b *Block) error {
	rs := b.state.dataState.RecordsState
	as := b.state.accState

	//Account state 변경 로직 추가
	if err := as.AddRecord(tx.owner, tx.recordHash); err != nil {
		return err
	}

	pbRecord := &corepb.Record{
		Owner:      tx.owner.Bytes(),
		RecordHash: tx.recordHash,
		Timestamp:  tx.timestamp,
	}
	recordBytes, err := proto.Marshal(pbRecord)
	if err != nil {
		return err
	}

	return rs.Put(tx.recordHash, recordBytes)
}

//Bandwidth returns bandwidth.
func (tx *AddRecordTx) Bandwidth() (*util.Uint128, error) {
	return TxBaseBandwidth, nil
}

//VestTx is a structure for withdrawing vesting
type VestTx struct {
	user   common.Address
	amount *util.Uint128
}

//NewVestTx returns NewTx
func NewVestTx(tx *Transaction) (ExecutableTx, error) {
	return &VestTx{
		user:   tx.From(),
		amount: tx.Value(),
	}, nil
}

//Execute VestTx
func (tx *VestTx) Execute(b *Block) error {
	as := b.state.AccState()

	if err := as.SubBalance(tx.user, tx.amount); err != nil {
		return err
	}
	if err := as.AddVesting(tx.user, tx.amount); err != nil {
		return err
	}

	account, err := as.GetAccount(tx.user)
	if err != nil {
		return err
	}

	// Add user's vesting to candidates' votePower
	iter, err := account.Voted.Iterator(nil)
	if err != nil {
		return err
	}

	exist, err := iter.Next()
	if err != nil {
		return err
	}
	for exist {
		addr := common.BytesToAddress(iter.Key())
		if err := as.AddVotePower(addr, tx.amount); err != nil {
			return err
		}

		exist, err = iter.Next()
		if err != nil {
			return err
		}
	}

	return nil
}

//Bandwidth returns bandwidth.
func (tx *VestTx) Bandwidth() (*util.Uint128, error) {
	return TxBaseBandwidth, nil
}

//WithdrawVestingTx is a structure for withdrawing vesting
type WithdrawVestingTx struct {
	user   common.Address
	amount *util.Uint128
}

//NewWithdrawVestingTx returns WithdrawVestingTx
func NewWithdrawVestingTx(tx *Transaction) (ExecutableTx, error) {
	return &WithdrawVestingTx{
		user:   tx.From(),
		amount: tx.Value(),
	}, nil
}

//Execute WithdrawVestingTx
func (tx *WithdrawVestingTx) Execute(b *Block) error {
	as := b.state.AccState()

	account, err := as.GetAccount(tx.user)
	if err != nil {
		return err
	}

	if err := as.SubVesting(tx.user, tx.amount); err != nil {
		return err
	}

	splitAmount, err := tx.amount.Div(util.NewUint128FromUint(RtWithdrawNum))
	if err != nil {
		return err
	}

	amountLeft := tx.amount.DeepCopy()

	payload := new(RtWithdraw)
	for i := 0; i < RtWithdrawNum; i++ {
		if i == RtWithdrawNum-1 {
			payload, err = NewRtWithdraw(amountLeft)
			if err != nil {
				return err
			}
		} else {
			payload, err = NewRtWithdraw(splitAmount)
			if err != nil {
				return err
			}
		}
		task := NewReservedTask(RtWithdrawType, tx.user, payload, b.Timestamp()+int64(i+1)*RtWithdrawInterval)
		if err := b.state.AddReservedTask(task); err != nil {
			return err
		}
		amountLeft, _ = amountLeft.Sub(splitAmount)
	}

	// Add user's vesting to candidates' votePower
	iter, err := account.Voted.Iterator(nil)
	if err != nil {
		return err
	}

	exist, err := iter.Next()
	if err != nil {
		return err
	}
	for exist {
		addr := common.BytesToAddress(iter.Key())
		if err := as.SubVotePower(addr, tx.amount); err != nil {
			return err
		}

		exist, err = iter.Next()
		if err != nil {
			return err
		}
	}
	return nil
}

//Bandwidth returns bandwidth.
func (tx *WithdrawVestingTx) Bandwidth() (*util.Uint128, error) {
	return TxBaseBandwidth, nil
}

//AddCertificationTx is a structure for adding certification
type AddCertificationTx struct {
	Issuer    common.Address
	Certified common.Address
	Payload   *AddCertificationPayload
}

//NewAddCertificationTx returns AddCertificationTx
func NewAddCertificationTx(tx *Transaction) (ExecutableTx, error) {
	payload := new(AddCertificationPayload)
	if err := BytesToTransactionPayload(tx.payload, payload); err != nil {
		return nil, err
	}

	//TODO: certification payload Verify: drsleepytiger
	return &AddCertificationTx{
		Issuer:    tx.From(),
		Certified: tx.To(),
		Payload:   payload,
	}, nil
}

//Execute AddCertificationTx
func (tx *AddCertificationTx) Execute(b *Block) error {
	as := b.state.AccState()
	cs := b.state.dataState.CertificationState

	if err := as.AddCertReceived(tx.Certified, tx.Payload.CertificateHash); err != nil {
		return err
	}
	if err := as.AddCertIssued(tx.Issuer, tx.Payload.CertificateHash); err != nil {
		return err
	}
	pbCertification := &corepb.Certification{
		CertificateHash: tx.Payload.CertificateHash,
		Issuer:          tx.Issuer.Bytes(),
		Certified:       tx.Certified.Bytes(),
		IssueTime:       tx.Payload.IssueTime,
		ExpirationTime:  tx.Payload.ExpirationTime,
		RevocationTime:  int64(-1),
	}
	certificationBytes, err := proto.Marshal(pbCertification)
	if err != nil {
		return err
	}
	return cs.Put(tx.Payload.CertificateHash, certificationBytes)
}

//Bandwidth returns bandwidth.
func (tx *AddCertificationTx) Bandwidth() (*util.Uint128, error) {
	return TxBaseBandwidth, nil
}

//RevokeCertificationTx is a structure for revoking certification
type RevokeCertificationTx struct {
	Revoker        common.Address
	Hash           []byte
	RevocationTime int64
}

//NewRevokeCertificationTx returns RevokeCertificationTx
func NewRevokeCertificationTx(tx *Transaction) (ExecutableTx, error) {
	payload := new(RevokeCertificationPayload)
	if err := BytesToTransactionPayload(tx.payload, payload); err != nil {
		return nil, err
	}

	//TODO: certification payload Verify: drsleepytiger
	return &RevokeCertificationTx{
		Revoker:        tx.From(),
		Hash:           payload.CertificateHash,
		RevocationTime: tx.timestamp,
	}, nil
}

//Execute RevokeCertificationTx
func (tx *RevokeCertificationTx) Execute(b *Block) error {
	s := b.state
	//as := b.state.AccState()
	cs := b.state.dataState.CertificationState

	pbCert, err := s.dataState.Certification(tx.Hash)
	if err != nil {
		return nil
	}

	if common.BytesToAddress(pbCert.Issuer) != tx.Revoker {
		return ErrInvalidCertificationRevoker
	}
	if pbCert.RevocationTime > int64(-1) {
		return ErrCertAlreadyRevoked
	}
	if pbCert.ExpirationTime < tx.RevocationTime {
		return ErrCertAlreadyExpired
	}

	pbCert.RevocationTime = tx.RevocationTime

	newBytesCertification, err := proto.Marshal(pbCert)
	if err != nil {
		return err
	}

	return cs.Put(tx.Hash, newBytesCertification)
}

//Bandwidth returns bandwidth.
func (tx *RevokeCertificationTx) Bandwidth() (*util.Uint128, error) {
	return TxBaseBandwidth, nil
}
