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

package sync

import (
	"errors"
	"time"

	"github.com/medibloc/go-medibloc/core"
)

// SyncService messages
const (
	BaseSearch   = "sync_base_req"
	BlockRequest = "sync_block_req"
)

// Parameters related sync service
const (
	SimultaneousRequest        = 1
	DefaultResponseTimeLimit   = 2 * time.Second
	DefaultNumberOfRetry       = 5
	DefaultActiveDownloadLimit = 10
)

//BlockManager is interface of core.blockmanager.
type BlockManager interface {
	Start()
	BlockHashByHeight(height uint64) ([]byte, error)
	BlockByHeight(height uint64) (*core.Block, error)
	BlockByHash(hash []byte) *core.Block
	LIB() *core.Block
	PushBlockDataSync(bd *core.BlockData, timeLimit time.Duration) error
}

// Error types
var (
	ErrCannotFindQueryID   = errors.New("cannot find query id on subscribe map")
	ErrContextDone         = errors.New("context is closed")
	ErrDifferentTargetHash = errors.New("target hash is different")
	ErrDownloadActivated   = errors.New("download is already activated")
	ErrFailedToConnect     = errors.New("failed to connect to peer")
	ErrLimitedRetry        = errors.New("retry is limited")
	ErrNotFound            = errors.New("not found")
	ErrWrongHeightBlock    = errors.New("peer send wrong block height")
)
