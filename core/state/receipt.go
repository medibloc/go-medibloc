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

package corestate

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// Receipt struct represents transaction receipt
type Receipt struct {
	executed  bool
	timestamp int64  // timestamp of block transaction included
	height    uint64 // height of block transaction included

	cpuUsage uint64
	netUsage uint64
	points   *util.Uint128

	error []byte
}

//Height returns height
func (r *Receipt) Height() uint64 {
	return r.height
}

//SetHeight sets height
func (r *Receipt) SetHeight(height uint64) {
	r.height = height
}

//Timestamp returns timestamp
func (r *Receipt) Timestamp() int64 {
	return r.timestamp
}

//SetTimestamp sets timestamp
func (r *Receipt) SetTimestamp(timestamp int64) {
	r.timestamp = timestamp
}

//CPUUsage returns cpu usage
func (r *Receipt) CPUUsage() uint64 {
	return r.cpuUsage
}

//SetCPUUsage sets cpu usage
func (r *Receipt) SetCPUUsage(cpuUsage uint64) {
	r.cpuUsage = cpuUsage
}

//NetUsage returns net usage
func (r *Receipt) NetUsage() uint64 {
	return r.netUsage
}

//SetNetUsage sets net usage
func (r *Receipt) SetNetUsage(netUsage uint64) {
	r.netUsage = netUsage
}

//Bandwidth returns bandwidth
func (r *Receipt) Bandwidth() *common.Bandwidth {
	return common.NewBandwidth(r.CPUUsage(), r.NetUsage())
}

//Points returns consumed points by transaction
func (r *Receipt) Points() *util.Uint128 {
	return r.points
}

//SetPoints sets points
func (r *Receipt) SetPoints(points *util.Uint128) {
	r.points = points
}

// SetError sets error occurred during transaction execution
func (r *Receipt) SetError(error []byte) {
	r.error = error
}

// Error returns error
func (r *Receipt) Error() []byte {
	return r.error
}

// SetExecuted sets transaction execution status
func (r *Receipt) SetExecuted(executed bool) {
	r.executed = executed
}

// Executed returns cpuPoints
func (r *Receipt) Executed() bool {
	return r.executed
}

// ToProto transform receipt struct to proto message
func (r *Receipt) ToProto() (proto.Message, error) {
	points, err := r.points.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	return &corepb.Receipt{
		Executed:  r.executed,
		Timestamp: r.timestamp,
		Height:    r.height,
		CpuUsage:  r.cpuUsage,
		NetUsage:  r.netUsage,
		Points:    points,
		Error:     r.error,
	}, nil
}

// FromProto transform receipt proto message to receipt struct
func (r *Receipt) FromProto(msg proto.Message) error {
	var err error
	if msg, ok := msg.(*corepb.Receipt); ok {
		r.executed = msg.Executed
		r.timestamp = msg.Timestamp
		r.height = msg.Height
		r.cpuUsage = msg.CpuUsage
		r.netUsage = msg.NetUsage
		r.points, err = util.NewUint128FromFixedSizeByteSlice(msg.Points)
		if err != nil {
			return err
		}
		r.error = msg.Error

		return nil
	}
	return ErrCannotConvertReceipt
}

func (r *Receipt) String() string {
	return fmt.Sprintf("{executed: %v, timestamp: %v, height: %v,cpu: %v, net: %v, points: %v, err: %v}", r.executed, r.timestamp, r.height, r.cpuUsage, r.netUsage, r.points.String(), r.error)
}

//Equal returns true if two receipts are equal
func (r *Receipt) Equal(obj *Receipt) bool {
	return r.executed == obj.executed &&
		r.timestamp == obj.timestamp &&
		r.height == obj.height &&
		r.cpuUsage == obj.cpuUsage &&
		r.netUsage == obj.netUsage &&
		r.points.Cmp(obj.points) == 0 &&
		byteutils.Equal(r.error, obj.error)
}

//NewReceipt returns new receipt
func NewReceipt() *Receipt {
	return &Receipt{
		executed:  false,
		timestamp: 0,
		height:    0,
		cpuUsage:  0,
		netUsage:  0,
		points:    util.NewUint128(),
		error:     nil,
	}
}
