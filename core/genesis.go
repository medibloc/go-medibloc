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
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
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

// TODO config verification, tx.verifyIntegrity, height, timestamp, chainid, ...
// NewGenesisBlock generates genesis block
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
		tx.SetReceipt(receipt)
		if err = genesis.AcceptTransaction(tx); err != nil {
			return nil, err
		}
		genesis.AppendTransaction(tx)
	}

	if err := genesis.Flush(); err != nil {
		return nil, err
	}

	return genesis, nil
}

// // NewGenesisBlock generates genesis block
// func NewGenesisBlock(conf *corepb.Genesis, consensus Consensus, stor storage.Storage) (*Block, error) {
// 	if conf == nil {
// 		return nil, ErrNilArgument
// 	}
//
// 	genesis, err := genesisTemplate(conf, consensus, stor)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if err := genesis.Prepare(); err != nil {
// 		return nil, err
// 	}
//
// 	// initialMessage := "Genesis block of MediBloc"
// 	// payload := &transaction.DefaultPayload{
// 	// 	Message: initialMessage,
// 	// }
// 	// payloadBuf, err := payload.ToBytes()
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
//
// 	initialTx := new(coreState.Transaction)
// 	// initialTx.SetTxType(TxTyGenesis)
// 	// initialTx.SetTo(GenesisCoinbase)
// 	// initialTx.SetValue(util.Uint128Zero())
// 	// initialTx.SetChainID(conf.Meta.ChainId)
// 	// initialTx.SetPayload(payloadBuf)
// 	// initialTx.SetReceipt(genesisTxReceipt())
//
// 	// txHash, err := initialTx.CalcHash()
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// initialTx.SetHash(txHash)
//
// 	// Genesis transactions do not consume bandwidth(only put in txState)
// 	if err := genesis.State().PutTx(initialTx); err != nil {
// 		return nil, err
// 	}
// 	genesis.AppendTransaction(initialTx) // append on block header
//
// 	// Token distribution
// 	supply := util.NewUint128()
// 	for _, dist := range conf.TokenDistribution {
// 		addr, err := common.HexToAddress(dist.Address)
// 		if err != nil {
// 			return nil, err
// 		}
// 		acc, err := genesis.State().GetAccount(addr)
// 		if err != nil {
// 			return nil, err
// 		}
// 		balance, err := util.NewUint128FromString(dist.Balance)
// 		if err != nil {
// 			if err := genesis.RollBack(); err != nil {
// 				return nil, err
// 			}
// 			return nil, err
// 		}
//
// 		acc.Balance = balance
// 		supply, err = supply.Add(balance)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		if err := genesis.State().PutAccount(acc); err != nil {
// 			return nil, err
// 		}
//
// 		tx := new(coreState.Transaction)
// 		// tx.SetTxType(TxTyGenesis)
// 		// tx.SetTo(addr)
// 		// tx.SetValue(acc.Balance)
// 		// tx.SetChainID(conf.Meta.ChainId)
// 		// tx.SetReceipt(genesisTxReceipt())
// 		// txHash, err := tx.CalcHash()
// 		// if err != nil {
// 		// 	return nil, err
// 		// }
// 		// tx.SetHash(txHash)
//
// 		// Genesis transactions do not consume bandwidth(only put in txState)
// 		if err := genesis.State().txState.Put(tx); err != nil {
// 			return nil, err
// 		}
// 		genesis.AppendTransaction(tx) // append on block header
//
// 	}
// 	genesis.State().supply = supply
//
// 	for _, pbTx := range conf.Transactions {
// 		tx := new(coreState.Transaction)
// 		if err := tx.FromProto(pbTx); err != nil {
// 			return nil, err
// 		}
// 		receipt, err := genesis.ExecuteTransaction(tx)
// 		if err != nil {
// 			return nil, err
// 		}
// 		tx.SetReceipt(receipt)
//
// 		if err = genesis.AcceptTransaction(tx); err != nil {
// 			return nil, err
// 		}
// 		genesis.AppendTransaction(tx) // append on block header
// 	}
//
// 	if err := genesis.Commit(); err != nil {
// 		return nil, err
// 	}
// 	if err := genesis.Flush(); err != nil {
// 		return nil, err
// 	}
//
// 	if err := genesis.Seal(); err != nil {
// 		return nil, err
// 	}
//
// 	return genesis, nil
// }

// CheckGenesisBlock checks if a block is genesis block
func CheckGenesisBlock(block *Block) bool {
	if block == nil {
		return false
	}
	return true
}

// CheckGenesisConf checks if block and genesis configuration match
func CheckGenesisConf(block *Block, genesis *corepb.Genesis) bool {
	if block.ChainID() != genesis.GetMeta().ChainId {
		logging.Console().WithFields(logrus.Fields{
			"block":   block,
			"genesis": genesis,
		}).Error("Genesis ChainID does not match.")
		return false
	}

	accounts, err := block.State().accState.Accounts()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get accounts from genesis block.")
		return false
	}

	tokenDist := genesis.GetTokenDistribution()
	if len(accounts) != len(tokenDist) {
		logging.Console().WithFields(logrus.Fields{
			"accountCount": len(accounts),
			"tokenCount":   len(tokenDist),
		}).Error("Size of token distribution accounts does not match.")
		return false
	}

	return true
}
