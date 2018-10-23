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

import "github.com/medibloc/go-medibloc/util"

// Receipt struct represents transaction receipt
type Receipt struct {
	executed       bool
	bandwidthUsage *util.Uint128
	error          []byte
}

// SetError sets error occurred during transaction execution
func (r *Receipt) SetError(error []byte) {
	r.error = error
}

// SetBandwidthUsage sets transaction's bandwidth
func (r *Receipt) SetBandwidthUsage(bandwidthUsage *util.Uint128) {
	r.bandwidthUsage = bandwidthUsage
}

// SetExecuted sets transaction execution status
func (r *Receipt) SetExecuted(executed bool) {
	r.executed = executed
}
