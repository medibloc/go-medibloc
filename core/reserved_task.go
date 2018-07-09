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
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"golang.org/x/crypto/sha3"
)

// ReservedTask is a data representing reserved task
type ReservedTask struct {
	taskType  string
	from      common.Address
	payload   Serializable
	timestamp int64
}

// NewReservedTask generates a new instance of ReservedTask
func NewReservedTask(taskType string, from common.Address, payload Serializable, timestamp int64) *ReservedTask {
	return &ReservedTask{
		taskType:  taskType,
		from:      from,
		payload:   payload,
		timestamp: timestamp,
	}
}

// ToProto converts ReservedTask to corepb.ReservedTask
func (t *ReservedTask) ToProto() (proto.Message, error) {
	payloadBytes, err := t.payload.Serialize()
	if err != nil {
		return nil, err
	}
	return &corepb.ReservedTask{
		Type:      t.taskType,
		From:      t.from.Bytes(),
		Payload:   payloadBytes,
		Timestamp: t.timestamp,
	}, nil
}

// FromProto converts
func (t *ReservedTask) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.ReservedTask); ok {
		t.taskType = msg.Type
		t.from = common.BytesToAddress(msg.From)

		var payload Serializable
		var err error
		switch msg.Type {
		case RtWithdrawType:
			payload, err = NewRtWithdraw(util.NewUint128())
		default:
			return ErrInvalidReservedTaskType
		}
		if err != nil {
			return err
		}

		if err := payload.Deserialize(msg.Payload); err != nil {
			return err
		}
		t.payload = payload
		t.timestamp = msg.Timestamp
		return nil
	}

	return ErrCannotConvertResevedTask
}

// TaskType returns t.taskType
func (t *ReservedTask) TaskType() string {
	return t.taskType
}

// From returns t.from
func (t *ReservedTask) From() common.Address {
	return t.from
}

// Payload returns t.payload
func (t *ReservedTask) Payload() Serializable {
	return t.payload
}

// Timestamp returns t.timestamp
func (t *ReservedTask) Timestamp() int64 {
	return t.timestamp
}

func (t *ReservedTask) calcHash() ([]byte, error) {
	hasher := sha3.New256()

	hasher.Write([]byte(t.taskType))
	hasher.Write(t.from.Bytes())
	payloadBytes, err := t.payload.Serialize()
	if err != nil {
		return nil, err
	}
	hasher.Write(payloadBytes)
	hasher.Write(byteutils.FromInt64(t.timestamp))

	return hasher.Sum(nil), nil
}

// ExecuteOnState following task's type and payload
func (t *ReservedTask) ExecuteOnState(bs *BlockState) error {
	switch t.taskType {
	case RtWithdrawType:
		return t.withdraw(bs)
	default:
		return ErrInvalidReservedTaskType
	}
}

func (t *ReservedTask) withdraw(bs *BlockState) error {
	amount := t.payload.(*RtWithdraw).Amount
	//if err := bs.SubVesting(t.from, amount); err != nil {
	//	return err
	//}
	return bs.AddBalance(t.from, amount)
}
