package core

import (
  "fmt"
  "github.com/golang/protobuf/proto"
  "github.com/medibloc/go-medibloc/common"
  "github.com/medibloc/go-medibloc/core/pb"
  "github.com/medibloc/go-medibloc/crypto"
  "github.com/medibloc/go-medibloc/keystore"
  "math/big"
)

type Transaction struct {
  hash    common.Hash
  from    common.Address
  to      common.Address
  value   *big.Int
  data    *corepb.Data
  chainID uint32
  alg     keystore.Algorithm
  sign    []byte
}

func NewTransaction(chainID uint32, from, to common.Address, value *big.Int, payloadType string, payload []byte) (*Transaction, error) {
  tx := &Transaction{
    from:    from,
    to:      to,
    value:   value,
    data:    &corepb.Data{Type: payloadType, Payload: payload},
    chainID: chainID,
    hash:    common.BytesToHash([]byte{}),
    sign:    []byte{},
  }

  return tx, nil
}

func (tx *Transaction) calcHash() (common.Hash, error) {
  hashTarget, err := tx.hashTargetBytes()
  if err != nil {
    var h common.Hash
    return h, ErrTransactionHashFailed
  }
  return common.BytesToHash(crypto.Sha3256(hashTarget)), nil
}

func (tx *Transaction) Sign(signer keystore.Signature) error {
  hash, err := tx.calcHash()
  if err != nil {
    return err
  }

  sig, err := signer.Sign(hash.Bytes())
  if err != nil {
    return err
  }
  tx.hash = hash
  tx.alg = signer.Algorithm()
  tx.sign = sig
  return nil
}

// VerifyIntegrity return transaction verify result, including Hash and Signature.
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

  pubKey, err := signature.RecoverPublic(tx.hash.Bytes(), tx.sign)
  if err != nil {
    return common.Address{}, err
  }

  return pubKey.Address()
}

func (tx *Transaction) From() common.Address {
  return tx.from
}

func (tx *Transaction) To() common.Address {
  return tx.to
}

func (tx *Transaction) Value() *big.Int {
  return tx.value
}

func (tx *Transaction) Type() string {
  return tx.data.Type
}

func (tx *Transaction) Data() []byte {
  return tx.data.Payload
}

func (tx *Transaction) ToProto() (proto.Message, error) {
  value := tx.value.Bytes()

  return &corepb.Transaction{
    Hash:    tx.hash.Bytes(),
    From:    tx.from.Bytes(),
    To:      tx.to.Bytes(),
    Value:   value,
    Data:    tx.data,
    ChainId: tx.chainID,
    Alg:     uint32(tx.alg),
    Sign:    tx.sign,
  }, nil
}

func (tx *Transaction) FromProto(msg proto.Message) error {
  if msg, ok := msg.(*corepb.Transaction); ok {
    tx.hash = common.BytesToHash(msg.Hash)
    tx.from = common.BytesToAddress(msg.From)
    tx.to = common.BytesToAddress(msg.To)
    tx.value = big.NewInt(0)
    tx.value.SetBytes(msg.Value)
    tx.data = msg.Data
    tx.chainID = msg.ChainId
    alg := keystore.Algorithm(msg.Alg)
    err := crypto.CheckAlgorithm(alg)
    if err != nil {
      return err
    }
    tx.alg = alg
    tx.sign = msg.Sign

    return nil
  }

  return ErrCannotConvertTransaction
}

func (tx *Transaction) hashTargetProto() proto.Message {
  return &corepb.TxHashTarget{
    From:    tx.from.Bytes(),
    To:      tx.to.Bytes(),
    Value:   tx.value.Bytes(),
    Data:    tx.data,
    Alg:     uint32(tx.alg),
    ChainId: tx.chainID,
  }
}

func (tx *Transaction) String() string {
  return fmt.Sprintf(`{"chainID":%d, "hash": "%x", "from": "%x", "to": "%x", "value":"%d", "type":"%s", "alg":"%d"}`,
    tx.chainID,
    tx.hash,
    tx.from,
    tx.to,
    tx.value.String(),
    tx.Type(),
    tx.alg,
  )
}

type Transactions []*Transaction

func (tx *Transaction) hashTargetBytes() ([]byte, error) {
  return proto.Marshal(tx.hashTargetProto())
}
