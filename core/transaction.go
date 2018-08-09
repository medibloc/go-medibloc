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

// ToProto converts Transaction to corepb.Transaction
func (t *Transaction) ToProto() (proto.Message, error) {
	value, err := t.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	return &corepb.Transaction{
		Hash:      t.hash,
		From:      t.from.Bytes(),
		To:        t.to.Bytes(),
		Value:     value,
		Timestamp: t.timestamp,
		Data:      t.data,
		Nonce:     t.nonce,
		ChainId:   t.chainID,
		Alg:       uint32(t.alg),
		Sign:      t.sign,
		PayerSign: t.payerSign,
	}, nil
}

// FromProto converts corepb.Transaction to Transaction
func (t *Transaction) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Transaction); ok {
		t.hash = msg.Hash
		t.from = common.BytesToAddress(msg.From)
		t.to = common.BytesToAddress(msg.To)

		value, err := util.NewUint128FromFixedSizeByteSlice(msg.Value)
		if err != nil {
			return err
		}
		t.value = value
		t.timestamp = msg.Timestamp
		t.data = msg.Data
		t.nonce = msg.Nonce
		t.chainID = msg.ChainId
		alg := algorithm.Algorithm(msg.Alg)
		err = crypto.CheckAlgorithm(alg)
		if err != nil {
			return err
		}
		t.alg = alg
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

//SetHash set hash
func (t *Transaction) SetHash(hash []byte) {
	t.hash = hash
}

//From returns from
func (t *Transaction) From() common.Address {
	return t.from
}

//SetFrom set from
func (t *Transaction) SetFrom(from common.Address) {
	t.from = from
}

//To returns to
func (t *Transaction) To() common.Address {
	return t.to
}

//SetTo set to
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

//Data returns data
func (t *Transaction) Data() *corepb.Data {
	return t.data
}

//SetData set data
func (t *Transaction) SetData(data *corepb.Data) {
	t.data = data
}

//Type returns type
func (t *Transaction) Type() string {
	return t.data.Type
}

//SetType set type
func (t *Transaction) SetType(ttype string) {
	t.data.Type = ttype
}

//Payload returns paylaod
func (t *Transaction) Payload() []byte {
	return t.data.Payload
}

//SetPayload set payload
func (t *Transaction) SetPayload(payload []byte) {
	t.data.Payload = payload
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

// Transactions is just multiple txs
type Transactions []*Transaction

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

// CalcHash calculates transaction's hash.
func (t *Transaction) CalcHash() ([]byte, error) {
	hasher := sha3.New256()

	value, err := t.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	txHashTarget := &corepb.TransactionHashTarget{
		From:      t.from.Bytes(),
		To:        t.to.Bytes(),
		Value:     value,
		Timestamp: t.timestamp,
		Data: &corepb.Data{
			Type:    t.data.Type,
			Payload: t.data.Payload,
		},
		Nonce:   t.nonce,
		ChainId: t.chainID,
	}
	data, err := proto.Marshal(txHashTarget)
	if err != nil {
		return nil, err
	}
	hasher.Write(data)

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
		t.Type(),
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
	*Transaction
}

//NewTransferTx returns TransferTx
func NewTransferTx(tx *Transaction) (ExecutableTx, error) {
	if tx.value.Cmp(util.Uint128Zero()) == 0 {
		return nil, ErrVoidTransaction
	}
	return &TransferTx{tx}, nil
}

//Execute TransferTx
func (tx *TransferTx) Execute(b *Block) error {
	as := b.state.AccState()

	if err := as.SubBalance(tx.from, tx.value); err != nil {
		return err
	}
	return as.AddBalance(tx.to, tx.value)
}

//AddRecordTx is a structure for adding record
type AddRecordTx struct {
	owner     common.Address
	timestamp int64
	payload   []byte
}

//NewAddRecordTx returns AddRecordTx
func NewAddRecordTx(tx *Transaction) (ExecutableTx, error) {
	return &AddRecordTx{
		owner:     tx.From(),
		timestamp: tx.Timestamp(),
		payload:   tx.Payload(),
	}, nil
}

//Execute AddRecordTx
func (tx *AddRecordTx) Execute(b *Block) error {
	rs := b.state.recordsState
	//as := b.state.accState

	// Account state 변경 로직 추가
	//as.AddRecord()

	var payload AddRecordPayload
	err := payload.FromBytes(tx.payload)
	if err != nil {
		return err
	}

	pbRecord := &corepb.Record{
		Hash:      payload.Hash,
		Owner:     tx.owner.Bytes(),
		Timestamp: tx.timestamp,
	}
	recordBytes, err := proto.Marshal(pbRecord)
	if err != nil {
		return err
	}

	return rs.Put(payload.Hash, recordBytes)
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
	candidates := account.Voted
	if candidates.RootHash() == nil {
		return nil
	}
	iter, err := candidates.Iterator(nil)
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
	candidates := account.Voted
	if candidates.RootHash() == nil {
		return nil
	}
	iter, err := candidates.Iterator(nil)
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

//AddCertificationTx is a structure for adding certification
type AddCertificationTx struct {
	Issuer    common.Address
	Certified common.Address
	Payload   *AddCertificationPayload
}

// AddCertificationPayload is payload type for AddCertificationTx
type AddCertificationPayload struct {
	IssueTime       int64
	ExpirationTime  int64
	CertificateHash []byte
}

//NewAddCertificationTx returns AddCertificationTx
func NewAddCertificationTx(tx *Transaction) (ExecutableTx, error) {
	payload := new(AddCertificationPayload)
	err := payload.FromBytes(tx.Payload())
	if err != nil {
		return nil, ErrInvalidTxPayload
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
	cs := b.state.certificationState

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

//RevokeCertificationTx is a structure for revoking certification
type RevokeCertificationTx struct {
	Revoker        common.Address
	Hash           []byte
	RevocationTime int64
}

// RevokeCertificationPayload is payload type for RevokeCertificationTx
type RevokeCertificationPayload struct {
	CertificateHash []byte
}

//NewRevokeCertificationTx returns RevokeCertificationTx
func NewRevokeCertificationTx(tx *Transaction) (ExecutableTx, error) {
	payload := new(RevokeCertificationPayload)
	if err := payload.FromBytes(tx.Payload()); err != nil {
		return nil, ErrInvalidTxPayload
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
	cs := b.state.certificationState

	pbCert, err := s.Certification(tx.Hash)
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
