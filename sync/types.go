package sync

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/common"
)

type BlockManager interface {
	BlockByHeight(height uint64) *core.Block
	BlockByHash(hash common.Hash) *core.Block
	LIB() *core.Block
	TailBlock() *core.Block
	PushBlockData(block *core.BlockData) error
}
