package sync

import (
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/common"
)

type BlockReceiver interface {
	Receive(block *core.Block)
}

type BlockGetter interface {
	Get(hash common.Hash) *core.Block
}

type BlockPoolWrap struct {
	pool *core.BlockPool
}

func NewBlockPoolWrap() (*BlockPoolWrap, error) {
	bp, err := core.NewBlockPool(128)
	if err != nil {
		return nil, err
	}
	return &BlockPoolWrap{
		pool: bp,
	}, nil
}

func (bp *BlockPoolWrap) Receive(block *core.Block) {
	bp.pool.Push(block)
}

func (bp *BlockPoolWrap) Get(hash *common.Hash) *core.Block {
	return bp.pool.Get(hash).(*core.Block)
}