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
)

// AddRecordPayload is payload type for TxOpAddRecord
type AddRecordPayload struct {
	RecordHash []byte
}

// FromBytes converts bytes to payload.
func (payload *AddRecordPayload) FromBytes(b []byte) error {
	if err := json.Unmarshal(b, payload); err != nil {
		return ErrInvalidTxPayload
	}
	return nil
}

// ToBytes returns marshaled AddRecordPayload
func (payload *AddRecordPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}

// AddCertificationPayload is payload type for AddCertificationTx
type AddCertificationPayload struct {
	IssueTime       int64
	ExpirationTime  int64
	CertificateHash []byte
}

// FromBytes converts bytes to payload.
func (payload *AddCertificationPayload) FromBytes(b []byte) error {
	if err := json.Unmarshal(b, payload); err != nil {
		return ErrInvalidTxPayload
	}
	return nil
}

// ToBytes returns marshaled AddCertificationPayload
func (payload *AddCertificationPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}

// RevokeCertificationPayload is payload type for RevokeCertificationTx
type RevokeCertificationPayload struct {
	CertificateHash []byte
}

// NewRevokeCertificationPayload generates a RevokeCertificationPayload
func NewRevokeCertificationPayload(hash []byte) *RevokeCertificationPayload {
	return &RevokeCertificationPayload{
		CertificateHash: hash,
	}
}

// FromBytes converts bytes to payload.
func (payload *RevokeCertificationPayload) FromBytes(b []byte) error {
	if err := json.Unmarshal(b, payload); err != nil {
		return ErrInvalidTxPayload
	}
	return nil
}

// ToBytes returns marshaled RevokeCertificationPayload
func (payload *RevokeCertificationPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}
