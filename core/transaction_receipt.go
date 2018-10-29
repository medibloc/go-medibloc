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
	"github.com/medibloc/go-medibloc/util"
)

// Receipt struct represents transaction receipt
type Receipt struct {
	executed bool
	cpuUsage *util.Uint128
	netUsage *util.Uint128
	error    []byte
}

// SetError sets error occurred during transaction execution
func (r *Receipt) SetError(error []byte) {
	r.error = error
}

// Error returns error
func (r *Receipt) Error() []byte {
	return r.error
}

// SetCPUUsage transaction's cpu bandwidth
func (r *Receipt) SetCPUUsage(cpuUsage *util.Uint128) {
	r.cpuUsage = cpuUsage
}

// CPUUsage returns cpuUsage
func (r *Receipt) CPUUsage() *util.Uint128 {
	return r.cpuUsage
}

// SetNetUsage sets transaction's net bandwidth
func (r *Receipt) SetNetUsage(netUsage *util.Uint128) {
	r.netUsage = netUsage
}

// NetUsage returns cpuUsage
func (r *Receipt) NetUsage() *util.Uint128 {
	return r.netUsage
}

// SetExecuted sets transaction execution status
func (r *Receipt) SetExecuted(executed bool) {
	r.executed = executed
}

// Executed returns cpuUsage
func (r *Receipt) Executed() bool {
	return r.executed
}

// ToProto transform receipt struct to proto message
func (r *Receipt) ToProto() (proto.Message, error) {
	cpuUsage, err := r.cpuUsage.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	netUsage, err := r.netUsage.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	return &corepb.Receipt{
		Executed: r.executed,
		CpuUsage: cpuUsage,
		NetUsage: netUsage,
		Error:    r.error,
	}, nil
}

// FromProto transform receipt proto message to receipt struct
func (r *Receipt) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Receipt); ok {
		cpuUsage, err := util.NewUint128FromFixedSizeByteSlice(msg.CpuUsage)
		if err != nil {
			return err
		}
		netUsage, err := util.NewUint128FromFixedSizeByteSlice(msg.NetUsage)
		if err != nil {
			return err
		}

		r.executed = msg.Executed
		r.cpuUsage = cpuUsage
		r.netUsage = netUsage
		r.error = msg.Error

		return nil
	}
	return ErrCannotConvertReceipt
}
