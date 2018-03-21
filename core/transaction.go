package core

import (
  "crypto/sha256"
  "fmt"
  "github.com/golang/protobuf/proto"
  "github.com/medibloc/go-medibloc/core/pb"
  "math/big"
)

type Transaction struct {
  hash    []byte
  from    []byte
  to      []byte
  value   *big.Int
  data    *corepb.Data
  chainID uint32
  sign    []byte
}

func (tx *Transaction) Hash() []byte {
  var hash [32]byte

  hashTarget := tx.HashTargetBytes()

  hash = sha256.Sum256(hashTarget)

  return hash[:]
}

func (tx *Transaction) From() []byte {
  return tx.from
}

func (tx *Transaction) To() []byte {
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
  value = tx.value.Bytes()

  return &corepb.Transaction{
    Hash:    tx.hash,
    From:    tx.from,
    To:      tx.to,
    Value:   value,
    Data:    tx.data,
    ChainId: tx.chainID,
    Sign:    tx.sign,
  }, nil
}

func (tx *Transaction) FromProto(msg proto.Message) error {
  if msg, ok := msg.(*corepb.Transaction); ok {
    tx.hash = msg.Hash
    tx.from = msg.From
    tx.to = msg.To
    tx.value = big.Int{}
    tx.value.SetBytes(msg.Value)
    tx.data = msg.Data
    tx.chainID = msg.ChainId
    tx.sign = msg.Sign

    return nil
  }

  return ErrCannotConvertTransaction
}

func (tx *Transaction) HashTargetProto() proto.Message {
  return &corepb.TxHashTarget{
    From:    tx.from,
    To:      tx.to,
    Value:   value,
    Data:    tx.data,
    ChainId: tx.chainID,
  }, nil
}

func (tx *Transaction) String() string {
  return fmt.Sprintf(`{"chainID":%d, "hash": "%x", "from": "%x", "to": "%x", "value":"%d", "type":"%s"}`,
    tx.chainID,
    tx.hash,
    tx.from,
    tx.to,
    tx.value.String(),
    tx.Type(),
  )
}

type Transactions []*Transaction

func NewTransaction(chainID uint32, from, to []byte, value *big.Int, payloadType string, payload []byte) (*Transaction, error) {
  tx := &Transaction{
    from:    from,
    to:      to,
    value:   value,
    data:    &corepb.Data{Type: payloadType, Payload: payload},
    chainID: chainID,
    hash:    []byte{},
    sign:    []byte{},
  }

  return tx, nil
}

func (tx *Transaction) HashTargetBytes() []byte {
  return proto.Marshal(tx.HashTargetProto())
}
