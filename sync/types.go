package sync

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
)

//BlockManager is interface of core.blockmanager.
type BlockManager interface {
	BlockByHeight(height uint64) *core.Block
	BlockByHash(hash common.Hash) *core.Block
	LIB() *core.Block
	TailBlock() *core.Block
	PushBlockData(block *core.BlockData) error
}
