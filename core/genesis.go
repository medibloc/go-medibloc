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
	"bytes"
	"errors"
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
)

var (
	// GenesisParentHash is hash of genesis block's parent hash
	GenesisParentHash = make([]byte, common.HashLength)

	// GenesisTimestamp is timestamp of genesis block
	GenesisTimestamp = int64(0)

	// GenesisCoinbase coinbase address of genesis block
	GenesisCoinbase, _ = common.HexToAddress("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	// TODO Add trie's key prefix of address
	//GenesisCoinbase, _ = common.HexToAddress("000000000000000000000000000000000000000000000000000000000000000000")

	// GenesisHeight is height of genesis block
	GenesisHeight = uint64(1)
)

// LoadGenesisConf loads genesis conf file
func LoadGenesisConf(filePath string) (*corepb.Genesis, error) {
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content := string(buf)

	genesis := new(corepb.Genesis)
	if err := proto.UnmarshalText(content, genesis); err != nil {
		return nil, err
	}
	return genesis, nil
}

func genesisTemplate(chainID uint32, consensus Consensus, stor storage.Storage) (*Block, error) {
	bd := &BlockData{
		BlockHeader: &BlockHeader{
			hash:         nil,
			parentHash:   GenesisParentHash,
			accStateRoot: nil,
			txStateRoot:  nil,
			dposRoot:     nil,
			coinbase:     GenesisCoinbase,
			reward:       util.NewUint128(),
			supply:       util.NewUint128(),
			timestamp:    GenesisTimestamp,
			chainID:      chainID,
			sign:         nil,
			cpuPrice:     util.NewUint128(),
			cpuUsage:     0,
			netPrice:     util.NewUint128(),
			netUsage:     0,
		},
		transactions: nil,
		height:       GenesisHeight,
	}

	bs, err := NewBlockState(bd, consensus, stor)
	if err != nil {
		return nil, err
	}

	return &Block{
		BlockData: bd,
		state:     bs,
		sealed:    false,
	}, nil
}

// NewGenesisBlock generates genesis block
// TODO config verification, tx.verifyIntegrity, height, timestamp, chainid, ...
func NewGenesisBlock(conf *corepb.Genesis, consensus Consensus, stor storage.Storage) (*Block, error) {
	genesis, err := genesisTemplate(conf.Meta.ChainId, consensus, stor)
	if err != nil {
		return nil, err
	}

	if err := genesis.Prepare(); err != nil {
		return nil, err
	}

	for _, pbTx := range conf.Transactions {
		tx := new(Transaction)
		if err := tx.FromProto(pbTx); err != nil {
			return nil, err
		}
		receipt, err := genesis.ExecuteTransaction(tx)
		if err != nil {
			return nil, err
		}
		if !receipt.executed {
			// TODO receipt error : []byte -> error
			return nil, errors.New(string(receipt.Error()))
		}
		tx.SetReceipt(receipt)
		if err = genesis.AcceptTransaction(tx); err != nil {
			return nil, err
		}
		genesis.AppendTransaction(tx)
	}

	if err := genesis.Flush(); err != nil {
		return nil, err
	}

	if err := genesis.Seal(); err != nil {
		return nil, err
	}

	return genesis, nil
}

// CheckGenesisConfig checks if block and genesis configuration match
func CheckGenesisConfig(conf *corepb.Genesis, consensus Consensus, block *Block) bool {
	stor, err := storage.NewMemoryStorage()
	if err != nil {
		return false
	}
	genesisBlock, err := NewGenesisBlock(conf, consensus, stor)
	if err != nil {
		return false
	}
	genesisBytes, err := genesisBlock.ToBytes()
	if err != nil {
		return false
	}
	blockBytes, err := block.ToBytes()
	if err != nil {
		return false
	}
	if !bytes.Equal(genesisBytes, blockBytes) {
		return false
	}
	return true
}
