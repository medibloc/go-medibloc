package core

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// TransactionTestWrap wraps Transaction and exports setters.
type TransactionTestWrap struct {
	*Transaction
}

// Clone clone transaction
func (t *TransactionTestWrap) Clone() *TransactionTestWrap {
	var receipt *Receipt
	if t.receipt != nil {
		receipt = &Receipt{
			executed:  t.receipt.executed,
			timestamp: t.receipt.timestamp,
			height:    t.receipt.height,
			cpuUsage:  t.receipt.cpuUsage,
			netUsage:  t.receipt.netUsage,
			points:    t.receipt.points.DeepCopy(),
			error:     byteutils.CopyBytes(t.receipt.error),
		}
	}
	tx := &Transaction{
		hash:      byteutils.CopyBytes(t.hash),
		txType:    t.txType,
		to:        t.to,
		value:     t.value.DeepCopy(),
		nonce:     t.nonce,
		chainID:   t.chainID,
		payload:   byteutils.CopyBytes(t.payload),
		sign:      byteutils.CopyBytes(t.sign),
		payerSign: byteutils.CopyBytes(t.payerSign),
		receipt:   receipt,
		from:      t.from,
		payer:     t.payer,
	}
	return &TransactionTestWrap{Transaction: tx}
}

// SetHash sets hash
func (t *TransactionTestWrap) SetHash(hash []byte) {
	t.hash = hash
}

// SetTxType sets type
func (t *TransactionTestWrap) SetTxType(txType string) {
	t.txType = txType
}

// SetTo sets set
func (t *TransactionTestWrap) SetTo(to common.Address) {
	t.to = to
}

// SetValue sets value
func (t *TransactionTestWrap) SetValue(value *util.Uint128) {
	t.value = value
}

// SetNonce sets nonce
func (t *TransactionTestWrap) SetNonce(nonce uint64) {
	t.nonce = nonce
}

// SetChainID sets chainID
func (t *TransactionTestWrap) SetChainID(chainID uint32) {
	t.chainID = chainID
}

// SetPayload sets payload
func (t *TransactionTestWrap) SetPayload(payload []byte) {
	t.payload = payload
}

// SetSign sets sign
func (t *TransactionTestWrap) SetSign(sign []byte) {
	t.sign = sign
}

// SetPayer sets payer
func (t *TransactionTestWrap) SetPayer(payer common.Address) {
	t.payer = payer
}

// SetPayerSign sets payerSign
func (t *TransactionTestWrap) SetPayerSign(payerSign []byte) {
	t.payerSign = payerSign
}

// SetReceipt sets receipt
func (t *TransactionTestWrap) SetReceipt(receipt *Receipt) {
	t.receipt = receipt
}

// SetFrom sets from
func (t *TransactionTestWrap) SetFrom(from common.Address) {
	t.from = from
}
