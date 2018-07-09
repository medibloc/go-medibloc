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
	"github.com/medibloc/go-medibloc/storage"
)

// Errors
var (
	ErrNotFound = storage.ErrKeyNotFound
)

// Action represents operation types in BatchTrie
type Action int

// Action constants
const (
	Insert Action = iota
	Update
	Delete
)

// Entry entry for not committed changes
type Entry struct {
	action Action
	key    []byte
	old    []byte
	update []byte
}
