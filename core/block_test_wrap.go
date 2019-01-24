package core

import (
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/util"
)

type BlockTestWrap struct {
	*Block
}

func (b *BlockTestWrap) Clone() (*BlockTestWrap, error) {
	bd, err := b.Block.BlockData.Clone()
	if err != nil {
		return nil, err
	}

	bs, err := b.Block.state.Clone()
	if err != nil {
		return nil, err
	}

	return &BlockTestWrap{
		&Block{
			BlockData: bd,
			state:     bs,
			sealed:    b.Block.sealed,
		}}, nil
}

func (b *BlockTestWrap) InitChild(coinbase common.Address) (*BlockTestWrap, error) {
	bb, err := b.Block.InitChild(coinbase)
	if err != nil {
		return nil, err
	}
	return &BlockTestWrap{Block: bb}, nil
}

// SetSealed set sealed
func (b *BlockTestWrap) SetSealed(sealed bool) {
	b.sealed = sealed
}

// SetHeight sets height.
func (b *BlockTestWrap) SetHeight(height uint64) {
	b.height = height
}

// SetTransactions sets transactions TO BE REMOVED: For test without block pool
func (b *BlockTestWrap) SetTransactions(txs []*Transaction) error {
	b.transactions = txs
	return nil
}

// SetHash set block header's hash
func (b *BlockTestWrap) SetHash(hash []byte) {
	b.hash = hash
}

// SetParentHash set block header's parent hash
func (b *BlockTestWrap) SetParentHash(parentHash []byte) {
	b.parentHash = parentHash
}

// SetAccStateRoot set block header's accStateRoot
func (b *BlockTestWrap) SetAccStateRoot(accStateRoot []byte) {
	b.accStateRoot = accStateRoot
}

// SetTxStateRoot set block header's txsRoot
func (b *BlockTestWrap) SetTxStateRoot(txStateRoot []byte) {
	b.txStateRoot = txStateRoot
}

// SetDposRoot set block header's dposRoot
func (b *BlockTestWrap) SetDposRoot(dposRoot []byte) {
	b.dposRoot = dposRoot
}

// SetCoinbase set coinbase
func (b *BlockTestWrap) SetCoinbase(coinbase common.Address) {
	b.coinbase = coinbase
}

// SetReward sets reward
func (b *BlockTestWrap) SetReward(reward *util.Uint128) {
	b.reward = reward
}

// SetSupply sets supply
func (b *BlockTestWrap) SetSupply(supply *util.Uint128) {
	b.supply = supply
}

// SetTimestamp sets timestamp of block
func (b *BlockTestWrap) SetTimestamp(timestamp int64) {
	b.timestamp = timestamp
}

// SetChainID sets chainID
func (b *BlockTestWrap) SetChainID(chainID uint32) {
	b.chainID = chainID
}

// SetCPUPrice sets cpuPrice
func (b *BlockTestWrap) SetCPUPrice(cpuPrice *util.Uint128) {
	b.cpuPrice = cpuPrice
}

// SetNetPrice sets netPrice
func (b *BlockTestWrap) SetNetPrice(netPrice *util.Uint128) {
	b.netPrice = netPrice
}

// SetSign sets sign
func (b *BlockTestWrap) SetSign(sign []byte) {
	b.sign = sign
}
