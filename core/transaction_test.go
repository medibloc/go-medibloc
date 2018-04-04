package core

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/keystore"
	"github.com/medibloc/go-medibloc/util"
	"github.com/stretchr/testify/assert"
)

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

func mockAddress(t *testing.T, ks *keystore.KeyStore) common.Address {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	assert.NoError(t, err)
	acc, err := ks.SetKey(privKey)
	assert.NoError(t, err)
	return acc
}

func TestTransaction(t *testing.T) {
	type fields struct {
		hash  common.Hash
		from  common.Address
		to    common.Address
		value *util.Uint128
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
				util.Uint128Zero(),
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
		privKey signature.PrivateKey
		count   int
	}

	tests := []testTx{}
	ks := keystore.NewKeyStore()

	for index := 0; index < testCount; index++ {

		from := mockAddress(t, ks)
		to := mockAddress(t, ks)

		key1, err := ks.GetKey(from)
		assert.NoError(t, err)

		tx, err := NewTransaction(1, from, to, util.Uint128Zero(), 1, TxPayloadBinaryType, []byte("datadata"))
		assert.NoError(t, err)

		sig, err := crypto.NewSignature(algorithm.SECP256K1)
		assert.NoError(t, err)
		sig.InitSign(key1)
		assert.NoError(t, tx.SignThis(sig))
		test := testTx{string(index), tx, key1, 1}
		tests = append(tests, test)
	}
	for _, tt := range tests {
		for index := 0; index < tt.count; index++ {
			t.Run(tt.name, func(t *testing.T) {
				signature, err := crypto.NewSignature(algorithm.SECP256K1)
				assert.NoError(t, err)
				signature.InitSign(tt.privKey)
				err = tt.tx.SignThis(signature)
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
