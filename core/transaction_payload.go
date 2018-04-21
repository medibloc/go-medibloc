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

// ToBytes returns marshaled RegisterWriterPayload
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

// ToBytes returns marshaled RemoveWriterPayload
func (payload *RemoveWriterPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}

// AddRecordPayload is payload type for TxOperationAddRecord
type AddRecordPayload struct {
	Hash common.Hash
	// TODO: Signature []byte
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

// AddRecordReaderPayload is payload type for TxOperationAddRecordReader
type AddRecordReaderPayload struct {
	Hash    common.Hash
	Address common.Address
	EncKey  []byte
	Seed    []byte
}

// NewAddRecordReaderPayload generates a AddRecordReaderPayload
func NewAddRecordReaderPayload(hash common.Hash, address common.Address, encKey, seed []byte) *AddRecordReaderPayload {
	return &AddRecordReaderPayload{
		Hash:    hash,
		Address: address,
		EncKey:  encKey,
		Seed:    seed,
	}
}

// BytesToAddRecordReaderPayload converts bytes to AddRecordReaderPayload struct
func BytesToAddRecordReaderPayload(b []byte) (*AddRecordReaderPayload, error) {
	payload := &AddRecordReaderPayload{}
	if err := json.Unmarshal(b, payload); err != nil {
		return nil, ErrInvalidTxPayload
	}
	return payload, nil
}

// ToBytes returns marshalled AddRecordReaderPayload
func (payload *AddRecordReaderPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}
