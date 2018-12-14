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
	"unicode"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/hash"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

const (
	//AliasKey key for find aliasname
	AliasKey = "alias"
	//MinimumAliasCollateral limit value for register alias
	MinimumAliasCollateral = "1000000"
)

// Transaction struct represents transaction
type Transaction struct {
	hash      []byte
	txType    string
	to        common.Address
	value     *util.Uint128
	timestamp int64
	nonce     uint64
	chainID   uint32
	payload   []byte
	hashAlg   algorithm.HashAlgorithm
	cryptoAlg algorithm.CryptoAlgorithm
	sign      []byte
	payerSign []byte

	receipt *Receipt

	from  common.Address
	payer common.Address
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
		To:        t.to.Bytes(),
		Value:     value,
		Timestamp: t.timestamp,
		Nonce:     t.nonce,
		ChainId:   t.chainID,
		Payload:   t.payload,
		HashAlg:   uint32(t.hashAlg),
		CryptoAlg: uint32(t.cryptoAlg),
		Sign:      t.sign,
		PayerSign: t.payerSign,
		Receipt:   Receipt,
	}, nil
}

// FromProto converts corepb.Transaction to Transaction
func (t *Transaction) FromProto(msg proto.Message) error {
	pbTx, ok := msg.(*corepb.Transaction)
	if !ok {
		return ErrCannotConvertTransaction
	}
	value, err := util.NewUint128FromFixedSizeByteSlice(pbTx.Value)
	if err != nil {
		return err
	}
	receipt := new(Receipt)
	if pbTx.Receipt != nil {
		if err := receipt.FromProto(pbTx.Receipt); err != nil {
			return err
		}
	} else {
		receipt = nil
	}

	t.hash = pbTx.Hash
	t.txType = pbTx.TxType
	t.to.FromBytes(pbTx.To)
	t.value = value
	t.timestamp = pbTx.Timestamp
	t.nonce = pbTx.Nonce
	t.chainID = pbTx.ChainId
	t.payload = pbTx.Payload
	t.hashAlg = algorithm.HashAlgorithm(pbTx.HashAlg)
	t.cryptoAlg = algorithm.CryptoAlgorithm(pbTx.CryptoAlg)
	t.sign = pbTx.Sign
	t.payerSign = pbTx.PayerSign
	t.receipt = receipt

	t.from, err = t.recoverSigner()
	if err == ErrTransactionSignatureNotExist {
		t.from = common.Address{}
	} else if err != nil {
		return err
	}
	t.payer, err = t.recoverPayer()
	if err == ErrPayerSignatureNotExist {
		t.payer = t.from
	} else if err != nil {
		return ErrCannotRecoverPayer
	}
	return nil
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

//HashAlg returns hashing algorithm
func (t *Transaction) HashAlg() algorithm.HashAlgorithm {
	return t.hashAlg
}

//SetHashAlg set hashing algorithm
func (t *Transaction) SetHashAlg(alg algorithm.HashAlgorithm) {
	t.hashAlg = alg
}

//CryptoAlg returns signing algorithm
func (t *Transaction) CryptoAlg() algorithm.CryptoAlgorithm {
	return t.cryptoAlg
}

//SetCryptoAlg set signing algorithm
func (t *Transaction) SetCryptoAlg(alg algorithm.CryptoAlgorithm) {
	t.cryptoAlg = alg
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

	return hash.GenHash(t.hashAlg, txHashTargetBytes)
}

// SignThis signs tx with given signature interface
func (t *Transaction) SignThis(key signature.PrivateKey) error {
	var err error
	t.from, err = common.PublicKeyToAddress(key.PublicKey())
	if err != nil {
		return err
	}

	t.hash, err = t.CalcHash()
	if err != nil {
		return err
	}

	signer, err := crypto.NewSignature(t.cryptoAlg)
	if err != nil {
		return err
	}
	signer.InitSign(key)

	t.sign, err = signer.Sign(t.hash)
	if err != nil {
		return err
	}
	return nil
}

func (t *Transaction) getPayerSignTarget() ([]byte, error) {
	payerSignTarget := &corepb.TransactionPayerSignTarget{
		Hash: t.Hash(),
		Sign: t.Sign(),
	}

	payerSignTargetBytes, err := proto.Marshal(payerSignTarget)
	if err != nil {
		return nil, err
	}

	return hash.Sha3256(payerSignTargetBytes), nil
}

func (t *Transaction) recoverPayer() (common.Address, error) {
	if t.payerSign == nil || len(t.payerSign) == 0 {
		return common.Address{}, ErrPayerSignatureNotExist
	}
	msg, err := t.getPayerSignTarget()
	if err != nil {
		return common.Address{}, err
	}

	if err := crypto.CheckCryptoAlgorithm(t.cryptoAlg); err != nil {
		return common.Address{}, err
	}

	sig, err := crypto.NewSignature(t.cryptoAlg)
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
	target, err := t.getPayerSignTarget()
	if err != nil {
		return err
	}

	sig, err := signer.Sign(target)
	if err != nil {
		return err
	}
	t.payerSign = sig
	return nil
}

// VerifyIntegrity returns transaction verify result, including Hash and Signature.
func (t *Transaction) VerifyIntegrity(chainID uint32) error {
	var err error
	// check ChainID.
	if t.chainID != chainID {
		return ErrInvalidChainID
	}

	// check Signature.
	if err := crypto.CheckCryptoAlgorithm(t.cryptoAlg); err != nil {
		return err
	}
	t.from, err = t.recoverSigner()
	if err != nil {
		return err
	}

	// check Hash.
	wantedHash, err := t.CalcHash()
	if err != nil {
		return err
	}
	if !byteutils.Equal(wantedHash, t.hash) {
		logging.Console().WithFields(logrus.Fields{
			"err":         err,
			"transaction": t,
		}).Warn("invalid tx hash")
		return ErrInvalidTransactionHash
	}

	return nil
}

func (t *Transaction) recoverSigner() (common.Address, error) {
	if t.sign == nil || len(t.sign) == 0 {
		return common.Address{}, ErrTransactionSignatureNotExist
	}

	if err := crypto.CheckCryptoAlgorithm(t.cryptoAlg); err != nil {
		return common.Address{}, err
	}

	sig, err := crypto.NewSignature(t.cryptoAlg)
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
	//fromStr := "no sign"
	//if t.sign != nil {
	//	from, err := t.recoverSigner()
	//	if err != nil {
	//		fromStr = "failed to recover signer"
	//	} else {
	//		fromStr = from.Hex()
	//	}
	//}

	return fmt.Sprintf(`{chainID:%v, hash:%v, from:%v, to:%v, value:%v, type:%v, cryptoAlg:%v, hashAlg:%v nonce:%v, 
timestamp:%v, receipt:%v}`,
		t.chainID,
		byteutils.Bytes2Hex(t.hash),
		t.from.Hex(),
		t.to.Hex(),
		t.value.String(),
		t.TxType(),
		t.cryptoAlg,
		t.hashAlg,
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

	newTx.from = t.from
	return newTx, nil
}

//Size returns bytes size of transaction
func (t *Transaction) Size() (int, error) {
	pbTx, err := t.ToProto()
	if err != nil {
		return 0, err
	}
	tmp, _ := pbTx.(*corepb.Transaction)
	tmp.Receipt = nil
	txBytes, err := proto.Marshal(tmp)
	if err != nil {
		return 0, err
	}
	return len(txBytes), nil
}

//TransferTx is a structure for sending MED
type TransferTx struct {
	from    common.Address
	to      common.Address
	value   *util.Uint128
	payload *DefaultPayload
	size    int
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
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}

	return &TransferTx{
		from:    tx.From(),
		to:      tx.To(),
		value:   tx.Value(),
		payload: payload,
		size:    size,
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
func (tx *TransferTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1000))
	if err != nil {
		return nil, nil, err
	}
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}

//AddRecordTx is a structure for adding record
type AddRecordTx struct {
	owner      common.Address
	timestamp  int64
	recordHash []byte
	size       int
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
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}

	return &AddRecordTx{
		owner:      tx.From(),
		timestamp:  tx.Timestamp(),
		recordHash: payload.RecordHash,
		size:       size,
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
func (tx *AddRecordTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1500))
	if err != nil {
		return nil, nil, err
	}
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}

//VestTx is a structure for withdrawing vesting
type VestTx struct {
	user   common.Address
	amount *util.Uint128
	size   int
}

//NewVestTx returns NewTx
func NewVestTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	if tx.Value().Cmp(util.Uint128Zero()) == 0 {
		return nil, ErrCannotUseZeroValue
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	return &VestTx{
		user:   tx.From(),
		amount: tx.Value(),
		size:   size,
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

	err = b.State().PutAccount(user)
	if err != nil {
		return err
	}

	voted := user.VotedSlice()

	// Add user's vesting to candidates' votePower
	for _, v := range voted {
		err = b.state.DposState().AddVotePowerToCandidate(v, tx.amount)
		if err != nil {
			return err
		}
	}
	return nil
}

//Bandwidth returns bandwidth.
func (tx *VestTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1000))
	if err != nil {
		return nil, nil, err
	}
	//fmt.Println("txSize:",uint64(tx.size))
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}

//WithdrawVestingTx is a structure for withdrawing vesting
type WithdrawVestingTx struct {
	user   common.Address
	amount *util.Uint128
	size   int
}

//NewWithdrawVestingTx returns WithdrawVestingTx
func NewWithdrawVestingTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.payload) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	return &WithdrawVestingTx{
		user:   tx.From(),
		amount: tx.Value(),
		size:   size,
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
		err = b.state.DposState().SubVotePowerToCandidate(v, tx.amount)
		if err != nil {
			return err
		}
	}
	return nil
}

//Bandwidth returns bandwidth.
func (tx *WithdrawVestingTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1000))
	if err != nil {
		return nil, nil, err
	}
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}

//AddCertificationTx is a structure for adding certification
type AddCertificationTx struct {
	Issuer          common.Address
	Certified       common.Address
	CertificateHash []byte
	IssueTime       int64
	ExpirationTime  int64
	size            int
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
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	return &AddCertificationTx{
		Issuer:          tx.From(),
		Certified:       tx.To(),
		CertificateHash: payload.CertificateHash,
		IssueTime:       payload.IssueTime,
		ExpirationTime:  payload.ExpirationTime,
		size:            size,
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
func (tx *AddCertificationTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1500))
	if err != nil {
		return nil, nil, err
	}
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}

//RevokeCertificationTx is a structure for revoking certification
type RevokeCertificationTx struct {
	Revoker         common.Address
	CertificateHash []byte
	RevocationTime  int64
	size            int
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
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	return &RevokeCertificationTx{
		Revoker:         tx.From(),
		CertificateHash: payload.CertificateHash,
		RevocationTime:  tx.timestamp,
		size:            size,
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
func (tx *RevokeCertificationTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1500))
	if err != nil {
		return nil, nil, err
	}
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}

// RegisterAliasTx is a structure for register alias
type RegisterAliasTx struct {
	addr       common.Address
	collateral *util.Uint128
	aliasName  string
	timestamp  int64
	size       int
}

//NewRegisterAliasTx returns RegisterAliasTx
func NewRegisterAliasTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(RegisterAliasPayload)
	if err := BytesToTransactionPayload(tx.payload, payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}

	return &RegisterAliasTx{
		addr:       tx.From(),
		aliasName:  payload.AliasName,
		collateral: tx.Value(),
		timestamp:  tx.Timestamp(),
		size:       size,
	}, nil
}

//Execute RegisterAliasTx
func (tx *RegisterAliasTx) Execute(b *Block) error {
	collateralLimit, err := util.NewUint128FromString(MinimumAliasCollateral)
	if err != nil {
		return err
	}
	if tx.collateral.Cmp(collateralLimit) < 0 {
		return ErrAliasCollateralLimit
	}

	err = checkAliasCondition(tx.aliasName)
	if err != nil {
		return err
	}
	acc, err := b.State().GetAccount(tx.addr)
	if err != nil {
		return err
	}
	//aliasBytes, err := acc.GetData(AliasPrefix, []byte("alias"))
	aliasBytes, err := acc.GetData(AliasPrefix, []byte(AliasKey))
	if err != nil && err != ErrNotFound {
		return err
	}
	pbAlias := new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		return err
	}
	if pbAlias.AliasName != "" {
		return ErrAlreadyHaveAlias
	}

	acc.Balance, err = acc.Balance.Sub(tx.collateral)
	if err == util.ErrUint128Underflow {
		return ErrBalanceNotEnough
	}
	if err != nil {
		return err
	}

	collateralBytes, err := tx.collateral.ToFixedSizeByteSlice()
	if err != nil {
		return err
	}
	pbAlias = &corepb.Alias{
		AliasName:       tx.aliasName,
		AliasCollateral: collateralBytes,
		Timestamp:       tx.timestamp,
	}
	aliasBytes, err = proto.Marshal(pbAlias)
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
	err = acc.PutData(AliasPrefix, []byte(AliasKey), aliasBytes)
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
	err = b.State().PutAccount(acc)
	if err != nil {
		return err
	}
	_, err = b.State().accState.GetAliasAccount(tx.aliasName)
	if err != nil && err != ErrNotFound {
		return err
	} else if err == nil {
		return ErrAliasAlreadyTaken
	}
	aa, err := newAliasAccount()
	if err != nil {
		return err
	}
	aa.Account = tx.addr
	b.State().accState.PutAliasAccount(aa, tx.aliasName)
	return nil
}

//Bandwidth returns bandwidth.
func (tx *RegisterAliasTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1500))
	if err != nil {
		return nil, nil, err
	}
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}

func checkAliasCondition(an string) error {
	if an == "" {
		return ErrAliasEmptyString
	}
	if len(an) > 12 {
		return ErrAliasLengthLimit
	}
	for i := 0; i < len(an); i++ {
		ch := rune(an[i])

		if !(unicode.IsNumber(ch) || !unicode.IsUpper(ch)) {
			return ErrAliasInvalidChar
		}
		if i == 0 && unicode.IsNumber(ch) {
			return ErrAliasFirsLetter
		}
	}
	return nil
}

// DeregisterAliasTx is a structure for deregister alias
type DeregisterAliasTx struct {
	addr common.Address
	size int
}

//NewDeregisterAliasTx returns RegisterAliasTx
func NewDeregisterAliasTx(tx *Transaction) (ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}

	return &DeregisterAliasTx{
		addr: tx.From(),
		size: size,
	}, nil
}

//Execute DeregisterAliasTx
func (tx *DeregisterAliasTx) Execute(b *Block) error {
	acc, err := b.State().GetAccount(tx.addr)
	if err != nil {
		return err
	}

	aliasBytes, err := acc.GetData(AliasPrefix, []byte(AliasKey))
	pbAlias := new(corepb.Alias)
	err = proto.Unmarshal(aliasBytes, pbAlias)
	if err != nil {
		return err
	}
	if pbAlias.AliasName == "" {
		return ErrAliasNotExist
	}
	collateral, err := util.NewUint128FromFixedSizeByteSlice(pbAlias.AliasCollateral)
	if err != nil {
		return err
	}
	acc.Balance, err = acc.Balance.Add(collateral)
	if err != nil {
		return err
	}

	err = b.State().accState.Delete([]byte(pbAlias.AliasName))
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
	err = acc.Data.Delete([]byte(AliasKey))
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
func (tx *DeregisterAliasTx) Bandwidth(bs *BlockState) (cpuUsage *util.Uint128, netUsage *util.Uint128, err error) {
	cpuUsage, err = bs.cpuRef.Mul(util.NewUint128FromUint(1500))
	if err != nil {
		return nil, nil, err
	}
	netUsage, err = bs.netRef.Mul(util.NewUint128FromUint(uint64(tx.size)))
	if err != nil {
		return nil, nil, err
	}
	return cpuUsage, netUsage, nil
}
