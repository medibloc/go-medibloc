package core

import (
	"crypto/ecdsa"
	"math/big"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/stretchr/testify/assert"
)

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

func mockNormalTransaction(t *testing.T, ks *keystore.KeyStore, chainID uint32, nonce uint64) *Transaction {
	return mockTransaction(t, ks, chainID, TxPayloadBinaryType, nil)
}

func mockAddress(t *testing.T, ks *keystore.KeyStore) common.Address {
	acc, err := ks.NewAccount("")
	assert.NoError(t, err)
	return acc
}

func mockTransaction(t *testing.T, ks *keystore.KeyStore, chainID uint32, payloadType string, payload []byte) *Transaction {
	from := mockAddress(t, ks)
	to := mockAddress(t, ks)
	tx, _ := NewTransaction(chainID, from, to, big.NewInt(0), payloadType, payload)
	return tx
}

func TestTransaction(t *testing.T) {
	type fields struct {
		hash  common.Hash
		from  common.Address
		to    common.Address
		value *big.Int
		data  *corepb.Data
	}
	ks := keystore.NewKeyStore()
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"full struct",
			fields(fields{
				common.BytesToHash([]byte("123455")),
				mockAddress(t, ks),
				mockAddress(t, ks),
				big.NewInt(0),
				&corepb.Data{Type: TxPayloadBinaryType, Payload: []byte("hwllo")},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{
				hash:  tt.fields.hash,
				from:  tt.fields.from,
				to:    tt.fields.to,
				value: tt.fields.value,
				data:  tt.fields.data,
			}
			msg, _ := tx.ToProto()
			ir, _ := proto.Marshal(msg)
			ntx := new(Transaction)
			nMsg := new(corepb.Transaction)
			proto.Unmarshal(ir, nMsg)
			ntx.FromProto(nMsg)
			if !reflect.DeepEqual(tx, ntx) {
				t.Errorf("Transaction.Serialize() = %v, want %v", *tx, *ntx)
			}
		})
	}
}

func TestTransaction_VerifyIntegrity(t *testing.T) {
	testCount := 3
	type testTx struct {
		name    string
		tx      *Transaction
		privKey *ecdsa.PrivateKey
		count   int
	}

	tests := []testTx{}
	ks := keystore.NewKeyStore()

	for index := 0; index < testCount; index++ {

		from := mockAddress(t, ks)
		to := mockAddress(t, ks)

		key1, err := ks.GetKey(from, "")
		assert.NoError(t, err)

		tx, err := NewTransaction(1, from, to, big.NewInt(0), TxPayloadBinaryType, []byte("datadata"))
		assert.NoError(t, err)

		assert.NoError(t, tx.Sign(key1.PrivateKey))
		test := testTx{string(index), tx, key1.PrivateKey, 1}
		tests = append(tests, test)
	}
	for _, tt := range tests {
		for index := 0; index < tt.count; index++ {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.tx.Sign(tt.privKey)
				if err != nil {
					t.Errorf("Sign() error = %v", err)
					return
				}
				err = tt.tx.VerifyIntegrity(tt.tx.chainID)
				if err != nil {
					t.Errorf("verify failed:%s", err)
					return
				}
			})
		}
	}
}
