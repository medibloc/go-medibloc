package core

import "github.com/medibloc/go-medibloc/common"

// TODO Remove when Genesis module implemented.
var TODOTestGenesisBlock = &Block{
	header: &BlockHeader{
		hash:       common.Hash{},
		parentHash: common.Hash{},
		coinbase:   common.Address{},
		timestamp:  0,
		chainID:    0,
		alg:        0,
		sign:       nil,
	},
	transactions: nil,
	sealed:       false,
	height:       0,
	parentBlock: &Block{
		header: &BlockHeader{
			hash:       common.Hash{},
			parentHash: common.Hash{},
			coinbase:   common.Address{},
			timestamp:  0,
			chainID:    0,
			alg:        0,
			sign:       nil,
		},
		transactions: nil,
		sealed:       false,
		height:       0,
		parentBlock:  &Block{},
	},
}
