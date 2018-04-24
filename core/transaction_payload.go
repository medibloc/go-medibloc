package core

import (
	"encoding/json"
)

// RegisterWriterPayload is payload type for TxOperationRegisterWKey tx
type RegisterWriterPayload struct {
	// Writer to register
	Writer string
}

// NewRegisterWriterPayload generates a RegisterWriterPayload value
func NewRegisterWriterPayload(writer string) *RegisterWriterPayload {
	return &RegisterWriterPayload{Writer: writer}
}

// BytesToRegisterWriterPayload converts bytes to RegisterWriterPayload struct
func BytesToRegisterWriterPayload(b []byte) (*RegisterWriterPayload, error) {
	payload := new(RegisterWriterPayload)
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
	Writer string
}

// NewRemoveWriterPayload generates a RemoveWriterPayload value
func NewRemoveWriterPayload(writer string) *RemoveWriterPayload {
	return &RemoveWriterPayload{Writer: writer}
}

// BytesToRemoveWriterPayload converts bytes to RemoveWriterPayload struct
func BytesToRemoveWriterPayload(b []byte) (*RemoveWriterPayload, error) {
	payload := new(RemoveWriterPayload)
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
	Hash string
	// TODO: Signature string
	Storage string
	EncKey  string
	Seed    string
}

// NewAddRecordPayload generates a AddRecordPayload
func NewAddRecordPayload(hash string, storage string, encKey, seed string) *AddRecordPayload {
	return &AddRecordPayload{
		Hash:    hash,
		Storage: storage,
		EncKey:  encKey,
		Seed:    seed,
	}
}

// BytesToAddRecordPayload converts bytes to AddRecordPayload struct
func BytesToAddRecordPayload(b []byte) (*AddRecordPayload, error) {
	payload := new(AddRecordPayload)
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
	Hash    string
	Address string
	EncKey  string
	Seed    string
}

// NewAddRecordReaderPayload generates a AddRecordReaderPayload
func NewAddRecordReaderPayload(hash, address, encKey, seed string) *AddRecordReaderPayload {
	return &AddRecordReaderPayload{
		Hash:    hash,
		Address: address,
		EncKey:  encKey,
		Seed:    seed,
	}
}

// BytesToAddRecordReaderPayload converts bytes to AddRecordReaderPayload struct
func BytesToAddRecordReaderPayload(b []byte) (*AddRecordReaderPayload, error) {
	payload := new(AddRecordReaderPayload)
	if err := json.Unmarshal(b, payload); err != nil {
		return nil, ErrInvalidTxPayload
	}
	return payload, nil
}

// ToBytes returns marshalled AddRecordReaderPayload
func (payload *AddRecordReaderPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}
