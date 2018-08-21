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
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/core/pb"
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

// AddCertificationPayload is payload type for AddCertificationTx
type AddCertificationPayload struct {
	IssueTime       int64
	ExpirationTime  int64
	CertificateHash []byte
}

// FromBytes converts bytes to payload.
func (payload *AddCertificationPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.AddCertificationPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.IssueTime = payloadPb.IssueTime
	payload.ExpirationTime = payloadPb.ExpirationTime
	payload.CertificateHash = payloadPb.Hash
	return nil
}

// ToBytes returns marshaled AddCertificationPayload
func (payload *AddCertificationPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.AddCertificationPayload{
		IssueTime:      payload.IssueTime,
		ExpirationTime: payload.ExpirationTime,
		Hash:           payload.CertificateHash,
	}
	return proto.Marshal(payloadPb)
}

// RevokeCertificationPayload is payload type for RevokeCertificationTx
type RevokeCertificationPayload struct {
	CertificateHash []byte
}

// FromBytes converts bytes to payload.
func (payload *RevokeCertificationPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.RevokeCertificationPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.CertificateHash = payloadPb.Hash
	return nil
}

// ToBytes returns marshaled RevokeCertificationPayload
func (payload *RevokeCertificationPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.RevokeCertificationPayload{
		Hash: payload.CertificateHash,
	}
	return proto.Marshal(payloadPb)
}

// DefaultPayload is payload type for any type of message
type DefaultPayload struct {
	Message string
}

// FromBytes converts bytes to payload.
func (payload *DefaultPayload) FromBytes(b []byte) error {
	payloadPb := &corepb.DefaultPayload{}
	if err := proto.Unmarshal(b, payloadPb); err != nil {
		return err
	}
	payload.Message = payloadPb.Message
	return nil
}

// ToBytes returns marshaled DefaultPayload
func (payload *DefaultPayload) ToBytes() ([]byte, error) {
	payloadPb := &corepb.DefaultPayload{
		Message: payload.Message,
	}
	return proto.Marshal(payloadPb)
}
