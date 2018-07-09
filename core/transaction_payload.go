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

// AddRecordPayload is payload type for TxOperationAddRecord
type AddRecordPayload struct {
	Hash []byte
}

// NewAddRecordPayload generates a AddRecordPayload
func NewAddRecordPayload(hash []byte) *AddRecordPayload {
	return &AddRecordPayload{
		Hash: hash,
	}
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

// NewAddCertificationPayload generates a AddCertificationPayload
func NewAddCertificationPayload(issueTime int64, expirationTime int64, certHash []byte) *AddCertificationPayload {
	return &AddCertificationPayload{
		IssueTime:       issueTime,
		ExpirationTime:  expirationTime,
		CertificateHash: certHash,
	}
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
