package core

import (
	"encoding/json"

	"github.com/medibloc/go-medibloc/common"
)

// RegisterWriterPayload is payload type for TxOperationRegisterWKey tx
type RegisterWriterPayload struct {
	// Writer to register
	Writer common.Address
}

// NewRegisterWriterPayload generates a RegisterWriterPayload value
func NewRegisterWriterPayload(writer common.Address) *RegisterWriterPayload {
	return &RegisterWriterPayload{Writer: writer}
}

// BytesToRegisterWriterPayload converts bytes to RegisterWriterPayload struct
func BytesToRegisterWriterPayload(b []byte) (*RegisterWriterPayload, error) {
	payload := &RegisterWriterPayload{}
	if err := json.Unmarshal(b, payload); err != nil {
		return nil, ErrInvalidTxPayload
	}
	return payload, nil
}

// ToBytes returns marshalled RegisterWriterPayload
func (payload *RegisterWriterPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}

// RemoveWriterPayload is payload type for TxOperationRemoveWKey tx
type RemoveWriterPayload struct {
	// Writer to remove
	Writer common.Address
}

// NewRemoveWriterPayload generates a RemoveWriterPayload value
func NewRemoveWriterPayload(writer common.Address) *RemoveWriterPayload {
	return &RemoveWriterPayload{Writer: writer}
}

// BytesToRemoveWriterPayload converts bytes to RemoveWriterPayload struct
func BytesToRemoveWriterPayload(b []byte) (*RemoveWriterPayload, error) {
	payload := &RemoveWriterPayload{}
	if err := json.Unmarshal(b, payload); err != nil {
		return nil, ErrInvalidTxPayload
	}
	return payload, nil
}

// ToBytes returns marshalled RemoveWriterPayload
func (payload *RemoveWriterPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}

// AddRecordPayload is payload type for TxOperationAddRecord
type AddRecordPayload struct {
	Hash    common.Hash
	Storage string
	EncKey  []byte
	Seed    []byte
}

// NewAddRecordPayload generates a AddRecordPayload
func NewAddRecordPayload(hash common.Hash, storage string, encKey, seed []byte) *AddRecordPayload {
	return &AddRecordPayload{
		Hash:    hash,
		Storage: storage,
		EncKey:  encKey,
		Seed:    seed,
	}
}

// BytesToAddRecordPayload converts bytes to AddRecordPayload struct
func BytesToAddRecordPayload(b []byte) (*AddRecordPayload, error) {
	payload := &AddRecordPayload{}
	if err := json.Unmarshal(b, payload); err != nil {
		return nil, ErrInvalidTxPayload
	}
	return payload, nil
}

// ToBytes returns marshalled AddRecordPayload
func (payload *AddRecordPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}
