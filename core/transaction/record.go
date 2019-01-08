package transaction

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// AddRecordPayload is payload type for TxOpAddRecord
type AddRecordPayload struct {
	RecordHash []byte
}

// FromBytes converts bytes to payload.
func (payload *AddRecordPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.AddRecordPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.RecordHash = payloadPb.Hash
	return nil
}

// ToBytes returns marshaled AddRecordPayload
func (payload *AddRecordPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.AddRecordPayload{
		Hash: payload.RecordHash,
	}
	return proto.Marshal(payloadPb)
}

//AddRecordTx is a structure for adding record
type AddRecordTx struct {
	owner      common.Address
	recordHash []byte
	size       int
}

//NewAddRecordTx returns AddRecordTx
func NewAddRecordTx(tx *coreState.Transaction) (*ExecutableTx, error) {
	if len(tx.Payload()) > MaxPayloadSize {
		return nil, ErrTooLargePayload
	}
	payload := new(AddRecordPayload)
	if err := BytesToTransactionPayload(tx.Payload(), payload); err != nil {
		return nil, err
	}
	size, err := tx.Size()
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(tx.From().Hex()) {
		return nil, ErrInvalidAddress
	}
	if !common.IsHash(byteutils.Bytes2Hex(payload.RecordHash)) {
		return nil, ErrRecordHashInvalid
	}

	return &ExecutableTx{
		Transaction: nil,
		Executable: &AddRecordTx{
			owner:      tx.From(),
			recordHash: payload.RecordHash,
			size:       size,
		},
	}, nil
}

//Execute AddRecordTx
func (tx *AddRecordTx) Execute(bs BlockState) error {
	var err error
	acc, err := bs.GetAccount(tx.owner)
	if err != nil {
		return err
	}

	_, err = acc.GetData(coreState.RecordsPrefix, tx.recordHash)
	if err != nil && err != ErrNotFound {
		return err
	}
	if err == nil {
		return ErrRecordAlreadyAdded
	}

	pbRecord := &corepb.Record{
		Owner:      tx.owner.Bytes(),
		RecordHash: tx.recordHash,
		Timestamp:  bs.Timestamp(),
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
	err = acc.PutData(coreState.RecordsPrefix, tx.recordHash, recordBytes)
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
	return bs.PutAccount(acc)
}

//Bandwidth returns bandwidth.
func (tx *AddRecordTx) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(1500, uint64(tx.size))
}
