// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

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

// NewRegisterWriterPayload generates a RegisterWriterPayload
func NewRegisterWriterPayload(writer common.Address) *RegisterWriterPayload {
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
	Writer common.Address
}

// NewRemoveWriterPayload generates a RemoveWriterPayload
func NewRemoveWriterPayload(writer common.Address) *RemoveWriterPayload {
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
	Hash []byte
	// TODO: Signature string
	Storage string
	EncKey  []byte
	Seed    []byte
}

// NewAddRecordPayload generates a AddRecordPayload
func NewAddRecordPayload(hash []byte) *AddRecordPayload {
	return &AddRecordPayload{
		Hash: hash,
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
