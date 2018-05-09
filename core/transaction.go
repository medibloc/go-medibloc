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
	hash      common.Hash
	from      common.Address
	to        common.Address
	value     *util.Uint128
	timestamp int64
	data      *corepb.Data
	nonce     uint64
	chainID   uint32
	alg       algorithm.Algorithm
	sign      []byte
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
		Hash:      tx.hash.Bytes(),
		From:      tx.from.Bytes(),
		To:        tx.to.Bytes(),
		Value:     value,
		Timestamp: tx.timestamp,
		Data:      tx.data,
		Nonce:     tx.nonce,
		ChainId:   tx.chainID,
		Alg:       uint32(tx.alg),
		Sign:      tx.sign,
	}, nil
}

// FromProto converts corepb.Transaction to Transaction
func (tx *Transaction) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Transaction); ok {
		tx.hash = common.BytesToHash(msg.Hash)
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
	hash common.Hash,
	alg uint32,
	sign []byte) (*Transaction, error) {
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
	return NewTransactionWithSign(chainID, from, to, value, nonce, payloadType, payload, common.BytesToHash([]byte{}), []byte{})
}

// NewTransactionWithSign generates a Transaction instance with sign
func NewTransactionWithSign(
	chainID uint32,
	from, to common.Address,
	value *util.Uint128,
	nonce uint64,
	payloadType string,
	payload []byte,
	hash common.Hash,
	sign []byte) (*Transaction, error) {

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
	}, nil
}

func (tx *Transaction) calcHash() (common.Hash, error) {
	hasher := sha3.New256()

	value, err := tx.value.ToFixedSizeByteSlice()
	if err != nil {
		return common.Hash{}, err
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
	return common.BytesToHash(hash), nil
}

// SignThis signs tx with given signature interface
func (tx *Transaction) SignThis(signer signature.Signature) error {
	tx.alg = signer.Algorithm()
	hash, err := tx.calcHash()
	if err != nil {
		return err
	}

	sig, err := signer.Sign(hash.Bytes())
	if err != nil {
		return err
	}
	tx.hash = hash
	tx.sign = sig
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
	if wantedHash.Equals(tx.hash) == false {
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
	if tx.Type() != TxOperationAddRecord && !tx.from.Equals(signer) {
		return ErrInvalidTransactionSigner
	}
	return nil
}

func (tx *Transaction) recoverSigner() (common.Address, error) {
	signature, err := crypto.NewSignature(tx.alg)
	if err != nil {
		return common.Address{}, err
	}

	pubKey, err := signature.RecoverPublic(tx.hash.Bytes(), tx.sign)
	if err != nil {
		return common.Address{}, err
	}

	return common.PublicKeyToAddress(pubKey)
}

// VerifyDelegation verifies if tx signer is equal to tx.from or one of tx.from's writers
func (tx *Transaction) VerifyDelegation(bs *BlockState) error {
	signer, err := tx.recoverSigner()
	if err != nil {
		return err
	}
	if byteutils.Equal(tx.from.Bytes(), signer.Bytes()) {
		return nil
	}
	acc, err := bs.GetAccount(tx.from)
	if err != nil {
		return err
	}
	for _, w := range acc.Writers() {
		if byteutils.Equal(signer.Bytes(), w) {
			return nil
		}
	}
	return ErrInvalidTxDelegation
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
func (tx *Transaction) Hash() common.Hash {
	return tx.hash
}

// Signature returns tx signature
func (tx *Transaction) Signature() []byte {
	return tx.sign
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
	case TxOperationRegisterWKey:
		return tx.registerWriteKey(bs)
	case TxOperationRemoveWKey:
		return tx.removeWriteKey(bs)
	case TxOperationAddRecord:
		return tx.addRecord(bs)
	case TxOperationAddRecordReader:
		return tx.addRecordReader(bs)
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

func (tx *Transaction) registerWriteKey(bs *BlockState) error {
	payload, err := BytesToRegisterWriterPayload(tx.Data())
	if err != nil {
		return err
	}
	return bs.AddWriter(tx.from, payload.Writer)
}

func (tx *Transaction) removeWriteKey(bs *BlockState) error {
	payload, err := BytesToRemoveWriterPayload(tx.Data())
	if err != nil {
		return err
	}
	return bs.RemoveWriter(tx.from, payload.Writer)
}

func (tx *Transaction) addRecord(bs *BlockState) error {
	signer, err := tx.recoverSigner()
	if err != nil {
		return err
	}
	payload, err := BytesToAddRecordPayload(tx.Data())
	if err != nil {
		return err
	}
	return bs.AddRecord(tx, payload.Hash, payload.Storage, payload.EncKey, payload.Seed, tx.from, signer)
}

func (tx *Transaction) addRecordReader(bs *BlockState) error {
	payload, err := BytesToAddRecordReaderPayload(tx.Data())
	if err != nil {
		return err
	}
	return bs.AddRecordReader(tx, payload.Hash, payload.Address, payload.EncKey, payload.Seed)
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
