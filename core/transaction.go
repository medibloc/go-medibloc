package core

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/hash"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/event"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// Transaction struct represents transaction
type Transaction struct {
	hash      []byte
	txType    string
	to        common.Address
	value     *util.Uint128
	nonce     uint64
	chainID   uint32
	payload   []byte
	sign      []byte
	payerSign []byte

	receipt *Receipt

	from  common.Address
	payer common.Address
}

type TransactionTemplateParam struct {
	TxType  string
	To      common.Address
	Value   *util.Uint128
	Nonce   uint64
	ChainID uint32
	Payload []byte
}

func NewTransactionTemplate(param *TransactionTemplateParam) *Transaction {
	return &Transaction{
		hash:      nil,
		txType:    param.TxType,
		to:        param.To,
		value:     param.Value,
		nonce:     param.Nonce,
		chainID:   param.ChainID,
		payload:   param.Payload,
		sign:      nil,
		payerSign: nil,
		receipt:   nil,
		from:      common.Address{},
		payer:     common.Address{},
	}
}

// Hash returns hash
func (t *Transaction) Hash() []byte {
	return t.hash
}

// TxType returns type
func (t *Transaction) TxType() string {
	return t.txType
}

// To returns to
func (t *Transaction) To() common.Address {
	return t.to
}

// Value returns value
func (t *Transaction) Value() *util.Uint128 {
	return t.value
}

// Nonce returns nonce
func (t *Transaction) Nonce() uint64 {
	return t.nonce
}

// ChainID returns chainID
func (t *Transaction) ChainID() uint32 {
	return t.chainID
}

// Payload returns payload
func (t *Transaction) Payload() []byte {
	return t.payload
}

// Sign returns sign
func (t *Transaction) Sign() []byte {
	return t.sign
}

// Payer returns payer
func (t *Transaction) Payer() common.Address {
	return t.payer
}

// PayerSign returns payerSign
func (t *Transaction) PayerSign() []byte {
	return t.payerSign
}

// Receipt returns receipt
func (t *Transaction) Receipt() *Receipt {
	return t.receipt
}

// SetReceipt sets receipt
func (t *Transaction) SetReceipt(receipt *Receipt) {
	t.receipt = receipt
}

func (t *Transaction) PayerOrFrom() common.Address {
	if t.payer.Equals(common.Address{}) {
		return t.from
	}
	return t.payer
}

func (t *Transaction) HasPayer() bool {
	return !t.payer.Equals(common.Address{})
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
		Nonce:     t.nonce,
		ChainId:   t.chainID,
		Payload:   t.payload,
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

	err = t.to.FromBytes(pbTx.To)
	if err != nil {
		return err
	}

	t.hash = pbTx.Hash
	t.txType = pbTx.TxType
	t.value = value
	t.nonce = pbTx.Nonce
	t.chainID = pbTx.ChainId
	t.payload = pbTx.Payload
	t.sign = pbTx.Sign
	t.payerSign = pbTx.PayerSign
	t.receipt = receipt

	t.from, err = t.recoverFrom()
	if err != nil {
		return err
	}
	t.payer, err = t.recoverPayer()
	if err != nil && err != ErrPayerSignatureNotExist {
		return ErrCannotRecoverPayer
	}
	return nil
}

// ToBytes convert transaction to
func (t *Transaction) ToBytes() ([]byte, error) {
	pb, err := t.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(pb)
}

// HexHash returns hex converted hash
func (t *Transaction) HexHash() string {
	return byteutils.Bytes2Hex(t.Hash())
}

// From returns from
func (t *Transaction) From() common.Address {
	return t.from
}

// CalcHash calculates transaction's hash.
func (t *Transaction) CalcHash() ([]byte, error) {
	value, err := t.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	txHashTarget := &corepb.TransactionHashTarget{
		TxType:  t.txType,
		From:    t.from.Bytes(),
		To:      t.to.Bytes(),
		Value:   value,
		Nonce:   t.nonce,
		ChainId: t.chainID,
		Payload: t.payload,
	}
	txHashTargetBytes, err := proto.Marshal(txHashTarget)
	if err != nil {
		return nil, err
	}

	return hash.GenHash(algorithm.SHA3256, txHashTargetBytes)
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

	signer, err := crypto.NewSignature(algorithm.SECP256K1)
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

func (t *Transaction) SignGenesis() error {
	var err error
	t.from = common.Address{}
	t.hash, err = t.CalcHash()
	if err != nil {
		return err
	}
	t.sign = nil
	return nil
}

func (t *Transaction) payerSignTarget() ([]byte, error) {
	target := &corepb.TransactionPayerSignTarget{
		Hash: t.hash,
		Sign: t.sign,
	}

	targetBytes, err := proto.Marshal(target)
	if err != nil {
		return nil, err
	}

	return hash.Sha3256(targetBytes), nil
}

func (t *Transaction) recoverPayer() (common.Address, error) {
	if t.payerSign == nil || len(t.payerSign) == 0 {
		return common.Address{}, ErrPayerSignatureNotExist
	}
	msg, err := t.payerSignTarget()
	if err != nil {
		return common.Address{}, err
	}

	sig, err := crypto.NewSignature(algorithm.SECP256K1)
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
	}).Debug("Secondary sign exist")
	return payer, nil
}

// SignByPayer puts payer's sign in tx
func (t *Transaction) SignByPayer(signer signature.Signature) error {
	target, err := t.payerSignTarget()
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
		return ErrInvalidTxChainID
	}

	t.from, err = t.recoverFrom()
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

func (t *Transaction) recoverFrom() (common.Address, error) {
	exeTx, err := TxConv(t)
	if err != nil {
		return common.Address{}, err
	}

	return exeTx.RecoverFrom()
}

// String returns string representation of tx
func (t *Transaction) String() string {
	return fmt.Sprintf(`{chainID:%v, hash:%v, from:%v, to:%v, value:%v, type:%v, nonce:%v, payload:%v, receipt:%v}`,
		t.chainID,
		byteutils.Bytes2Hex(t.hash),
		t.from.Hex(),
		t.to.Hex(),
		t.value.String(),
		t.txType,
		t.nonce,
		byteutils.Bytes2Hex(t.payload),
		t.receipt,
	)
}

// TriggerEvent triggers non account type event
func (t *Transaction) TriggerEvent(e *event.Emitter, eTopic string) {
	event := &event.Event{
		Topic: eTopic,
		Data:  byteutils.Bytes2Hex(t.hash),
		Type:  "",
	}
	e.Trigger(event)
	return
}

// TriggerAccEvent triggers account type event
func (t *Transaction) TriggerAccEvent(e *event.Emitter, eType string) {
	ev := &event.Event{
		Topic: t.From().String(),
		Data:  byteutils.Bytes2Hex(t.hash),
		Type:  eType,
	}
	e.Trigger(ev)

	if t.to.String() != "" {
		ev = &event.Event{
			Topic: t.to.String(),
			Data:  byteutils.Bytes2Hex(t.hash),
			Type:  eType,
		}
		e.Trigger(ev)
	}
	return
}

// Size returns bytes size of transaction
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
