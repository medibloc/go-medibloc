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

// AddCertificationPayload is payload type for TxOperationAddCertification
type AddCertificationPayload struct {
	IssueTime       int64
	ExpirationTime  int64
	CertificateHash []byte
}

// NewAddCertificationPayload generates a AddCertificationPayload
func NewAddCertificationPayload(issueTime int64, expirationTime int64, certHash []byte) *AddCertificationPayload {
	return &AddCertificationPayload{
		IssueTime:       issueTime,
		ExpirationTime:  expirationTime,
		CertificateHash: certHash,
	}
}

// BytesToAddCertificationPayload converts bytes to AddCertificationPayload struct
func BytesToAddCertificationPayload(b []byte) (*AddCertificationPayload, error) {
	payload := new(AddCertificationPayload)
	if err := json.Unmarshal(b, payload); err != nil {
		return nil, ErrInvalidTxPayload
	}
	return payload, nil
}

// ToBytes returns marshalled AddCertificationPayload
func (payload *AddCertificationPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}

// RevokeCertificationPayload is payload type for TxOperationRevokeCertification
type RevokeCertificationPayload struct {
	CertificateHash []byte
}

// NewRevokeCertificationPayload generates a RevokeCertificationPayload
func NewRevokeCertificationPayload(hash []byte) *RevokeCertificationPayload {
	return &RevokeCertificationPayload{
		CertificateHash: hash,
	}
}

// BytesToRevokeCertificationPayload converts bytes to RevokeCertificationPayload struct
func BytesToRevokeCertificationPayload(b []byte) (*RevokeCertificationPayload, error) {
	payload := new(RevokeCertificationPayload)
	if err := json.Unmarshal(b, payload); err != nil {
		return nil, ErrInvalidTxPayload
	}
	return payload, nil
}

// ToBytes returns marshalled RevokeCertificationPayload
func (payload *RevokeCertificationPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}
