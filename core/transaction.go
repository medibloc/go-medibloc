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
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

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

	receipt *Receipt
}

// ToProto converts Transaction to corepb.Transaction
func (t *Transaction) ToProto() (proto.Message, error) {
	value, err := t.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	var Receipt *corepb.Receipt
	if t.receipt != nil {
		receipt, err := t.receipt.ToProto()
		if err != nil {
			return nil, err
		}

		var ok bool
		Receipt, ok = receipt.(*corepb.Receipt)
		if !ok {
			return nil, ErrInvalidReceiptToProto
		}
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
		Receipt:   Receipt,
	}, nil
}

// FromProto converts corepb.Transaction to Transaction
func (t *Transaction) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Transaction); ok {
		value, err := util.NewUint128FromFixedSizeByteSlice(msg.Value)
		if err != nil {
			return err
		}

		receipt := new(Receipt)
		if msg.Receipt != nil {
			if err := receipt.FromProto(msg.Receipt); err != nil {
				return err
			}
		} else {
			receipt = nil
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
		t.receipt = receipt

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

//Receipt returns receipt
func (t *Transaction) Receipt() *Receipt {
	return t.receipt
}

//SetReceipt set receipt
func (t *Transaction) SetReceipt(receipt *Receipt) {
	t.receipt = receipt
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

	payer, err := common.PublicKeyToAddress(pubKey)
	if err != nil {
		return common.Address{}, err
	}
	logging.Console().WithFields(logrus.Fields{
		"payer": payer.Hex(),
	}).Info("Secondary sign exist")
	return payer, nil
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
	return fmt.Sprintf(`{chainID:%v, hash:%v, from:%v, to:%v, value:%v, type:%v, alg:%v, nonce:%v, timestamp:%v, 
receipt:%v}`,
		t.chainID,
		byteutils.Bytes2Hex(t.hash),
		t.from,
		t.to,
		t.value.String(),
		t.TxType(),
		t.alg,
		t.nonce,
		t.timestamp,
		t.receipt,
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
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
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
	// subtract balance from sender's account
	sender, err := b.state.GetAccount(tx.from)
	if err != nil {
		return err
	}
	sender.Balance, err = sender.Balance.Sub(tx.value)
	if err == util.ErrUint128Underflow {
		return ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}
	err = b.State().PutAccount(sender)
	if err != nil {
		return err
	}

	// add balance to receiver's account
	receiver, err := b.state.GetAccount(tx.to)
	if err != nil {
		return err
	}
	receiver.Balance, err = receiver.Balance.Add(tx.value)
	if err != nil {
		return err
	}
	err = b.State().PutAccount(receiver)
	if err != nil {
		return err
	}
	return nil
}

//Bandwidth returns bandwidth.
func (tx *TransferTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return TxBaseCPUBandwidth, TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}

//AddRecordTx is a structure for adding record
type AddRecordTx struct {
	owner      common.Address
	timestamp  int64
	recordHash []byte
}

//NewAddRecordTx returns AddRecordTx
func NewAddRecordTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
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
	var err error
	acc, err := b.State().GetAccount(tx.owner)
	if err != nil {
		return err
	}

	_, err = acc.GetData(RecordsPrefix, tx.recordHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrRecordAlreadyAdded
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
	err = acc.Data.Prepare()
	if err != nil {
		return err
	}
	err = acc.Data.BeginBatch()
	if err != nil {
		return err
	}
	err = acc.PutData(RecordsPrefix, tx.recordHash, recordBytes)
	if err != nil {
		return err
	}
	err = acc.Data.Commit()
	if err != nil {
		return err
	}
	err = acc.Data.Flush()
	if err != nil {
		return err
	}
	return b.State().PutAccount(acc)
}

//Bandwidth returns bandwidth.
func (tx *AddRecordTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return TxBaseCPUBandwidth, TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}

//VestTx is a structure for withdrawing vesting
type VestTx struct {
	user   common.Address
	amount *util.Uint128
}

//NewVestTx returns NewTx
func NewVestTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	if tx.Value().Cmp(util.Uint128Zero()) == 0 {
		return nil, ErrCannotUseZeroValue
	}
	return &VestTx{
		user:   tx.From(),
		amount: tx.Value(),
	}, nil
}

//Execute VestTx
func (tx *VestTx) Execute(b *Block) error {

	user, err := b.State().GetAccount(tx.user)
	if err != nil {
		return err
	}
	user.Balance, err = user.Balance.Sub(tx.amount)
	if err == util.ErrUint128Underflow {
		return ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}
	user.Vesting, err = user.Vesting.Add(tx.amount)
	if err != nil {
		return err
	}
	voted := user.VotedSlice()

	err = b.State().PutAccount(user)
	if err != nil {
		return err
	}

	// Add user's vesting to candidates' votePower
	for _, v := range voted {
		candidate, err := b.State().GetAccount(common.BytesToAddress(v))
		if err != nil {
			return err
		}
		candidate.VotePower, err = candidate.VotePower.Add(tx.amount)
		if err != nil {
			return err
		}
		err = b.State().PutAccount(candidate)
		if err != nil {
			return err
		}
	}
	return nil
}

//Bandwidth returns bandwidth.
func (tx *VestTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return TxBaseCPUBandwidth, TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}

//WithdrawVestingTx is a structure for withdrawing vesting
type WithdrawVestingTx struct {
	user   common.Address
	amount *util.Uint128
}

//NewWithdrawVestingTx returns WithdrawVestingTx
func NewWithdrawVestingTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	return &WithdrawVestingTx{
		user:   tx.From(),
		amount: tx.Value(),
	}, nil
}

//Execute WithdrawVestingTx
func (tx *WithdrawVestingTx) Execute(b *Block) error {
	account, err := b.State().GetAccount(tx.user)
	if err != nil {
		return err
	}

	if account.Vesting.Cmp(tx.amount) < 0 {
		return ErrVestingNotEnough
	}
	account.Vesting, err = account.Vesting.Sub(tx.amount)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to subtract vesting.")
		return err
	}

	account.Unstaking, err = account.Unstaking.Add(tx.amount)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to add unstaking.")
		return err
	}
	account.LastUnstakingTs = b.Timestamp()

	// Update bandwidth
	if tx.amount.Cmp(account.Bandwidth) >= 0 {
		account.Bandwidth = util.NewUint128()
	} else {
		account.Bandwidth, err = account.Bandwidth.Sub(tx.amount)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Warn("Failed to subtract bandwidth.")
			return err
		}
	}

	voted := account.VotedSlice()

	err = b.State().PutAccount(account)
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Warn("Failed to put account.")
		return err
	}

	// Add user's vesting to candidates' votePower
	for _, v := range voted {
		candidate, err := b.State().GetAccount(common.BytesToAddress(v))
		if err != nil {
			return err
		}
		candidate.VotePower, err = candidate.VotePower.Sub(tx.amount)
		if err != nil {
			return err
		}
		err = b.State().PutAccount(candidate)
		if err != nil {
			return err
		}
	}
	return nil
}

//Bandwidth returns bandwidth.
func (tx *WithdrawVestingTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return TxBaseCPUBandwidth, TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}

//AddCertificationTx is a structure for adding certification
type AddCertificationTx struct {
	Issuer    common.Address
	Certified common.Address

	CertificateHash []byte
	IssueTime       int64
	ExpirationTime  int64
}

//NewAddCertificationTx returns AddCertificationTx
func NewAddCertificationTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(AddCertificationPayload)
	if err := BytesToTransactionPayload(tx.payload, payload); err != nil {
		return nil, err
	}

	return &AddCertificationTx{
		Issuer:          tx.From(),
		Certified:       tx.To(),
		CertificateHash: payload.CertificateHash,
		IssueTime:       payload.IssueTime,
		ExpirationTime:  payload.ExpirationTime,
	}, nil
}

//Execute AddCertificationTx
func (tx *AddCertificationTx) Execute(b *Block) error {
	certified, err := b.State().GetAccount(tx.Certified)
	if err != nil {
		return err
	}
	_, err = certified.GetData(CertReceivedPrefix, tx.CertificateHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrCertReceivedAlreadyAdded
	}

	issuer, err := b.State().GetAccount(tx.Issuer)
	if err != nil {
		return err
	}
	_, err = issuer.GetData(CertIssuedPrefix, tx.CertificateHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrCertIssuedAlreadyAdded
	}

	//TODO: certification payload Verify: drsleepytiger

	pbCertification := &corepb.Certification{
		CertificateHash: tx.CertificateHash,
		Issuer:          tx.Issuer.Bytes(),
		Certified:       tx.Certified.Bytes(),
		IssueTime:       tx.IssueTime,
		ExpirationTime:  tx.ExpirationTime,
		RevocationTime:  int64(-1),
	}
	certificationBytes, err := proto.Marshal(pbCertification)
	if err != nil {
		return err
	}

	// Add certification to certified's account state
	if err := certified.Data.Prepare(); err != nil {
		return err
	}
	if err := certified.Data.BeginBatch(); err != nil {
		return err
	}
	if err := certified.PutData(CertReceivedPrefix, tx.CertificateHash, certificationBytes); err != nil {
		if err := certified.Data.RollBack(); err != nil {
			return err
		}
		return err
	}
	if err := certified.Data.Commit(); err != nil {
		return err
	}
	if err := certified.Data.Flush(); err != nil {
		return err
	}
	if err := b.State().PutAccount(certified); err != nil {
		return err
	}

	// Add certification to issuer's account state
	issuer, err = b.State().GetAccount(tx.Issuer)
	if err != nil {
		return err
	}
	if err := issuer.Data.Prepare(); err != nil {
		return err
	}
	if err := issuer.Data.BeginBatch(); err != nil {
		return err
	}
	if err := issuer.PutData(CertIssuedPrefix, tx.CertificateHash, certificationBytes); err != nil {
		if err := issuer.Data.RollBack(); err != nil {
			return err
		}
		return err
	}
	if err := issuer.Data.Commit(); err != nil {
		return err
	}
	if err := issuer.Data.Flush(); err != nil {
		return err
	}
	if err := b.State().PutAccount(issuer); err != nil {
		return err
	}

	return nil
}

//Bandwidth returns bandwidth.
func (tx *AddCertificationTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return TxBaseCPUBandwidth, TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}

//RevokeCertificationTx is a structure for revoking certification
type RevokeCertificationTx struct {
	Revoker         common.Address
	CertificateHash []byte
	RevocationTime  int64
}

//NewRevokeCertificationTx returns RevokeCertificationTx
func NewRevokeCertificationTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(RevokeCertificationPayload)
	if err := BytesToTransactionPayload(tx.payload, payload); err != nil {
		return nil, err
	}
	return &RevokeCertificationTx{
		Revoker:         tx.From(),
		CertificateHash: payload.CertificateHash,
		RevocationTime:  tx.timestamp,
	}, nil
}

//Execute RevokeCertificationTx
func (tx *RevokeCertificationTx) Execute(b *Block) error {
	issuer, err := b.State().GetAccount(tx.Revoker)
	if err != nil {
		return err
	}
	certBytes, err := issuer.GetData(CertIssuedPrefix, tx.CertificateHash)
	if err != nil {
		return err
	}

	pbCert := new(corepb.Certification)
	err = proto.Unmarshal(certBytes, pbCert)
	if err != nil {
		return err
	}
	// verify transaction
	if !byteutils.Equal(pbCert.Issuer, tx.Revoker.Bytes()) {
		return ErrInvalidCertificationRevoker
	}
	if pbCert.RevocationTime > int64(-1) {
		return ErrCertAlreadyRevoked
	}
	if pbCert.ExpirationTime < tx.RevocationTime {
		return ErrCertAlreadyExpired
	}

	pbCert.RevocationTime = tx.RevocationTime
	newCertBytes, err := proto.Marshal(pbCert)
	if err != nil {
		return err
	}
	// change cert on issuer's cert issued List
	err = issuer.Data.Prepare()
	if err != nil {
		return err
	}
	err = issuer.Data.BeginBatch()
	if err != nil {
		return err
	}
	err = issuer.PutData(CertIssuedPrefix, tx.CertificateHash, newCertBytes)
	if err != nil {
		return err
	}
	err = issuer.Data.Commit()
	if err != nil {
		return err
	}
	err = issuer.Data.Flush()
	if err != nil {
		return err
	}
	err = b.State().PutAccount(issuer)
	if err != nil {
		return err
	}
	// change cert on certified's cert received list
	certified, err := b.State().GetAccount(common.BytesToAddress(pbCert.Certified))
	if err != nil {
		return err
	}
	err = certified.Data.Prepare()
	if err != nil {
		return err
	}
	err = certified.Data.BeginBatch()
	if err != nil {
		return err
	}
	err = certified.PutData(CertReceivedPrefix, tx.CertificateHash, newCertBytes)
	if err != nil {
		return err
	}
	err = certified.Data.Commit()
	if err != nil {
		return err
	}
	err = certified.Data.Flush()
	if err != nil {
		return err
	}
	return b.State().PutAccount(certified)
}

//Bandwidth returns bandwidth.
func (tx *RevokeCertificationTx) Bandwidth() (*util.Uint128, *util.Uint128, error) {
	return TxBaseCPUBandwidth, TxBaseNetBandwidth, nil // TODO use cpu, net bandwidth
}
