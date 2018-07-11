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
	"github.com/medibloc/go-medibloc/consensus/dpos/pb"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

var (
	// GenesisHash is hash of genesis block
	GenesisHash = []byte("genesisHash")
	// GenesisTimestamp is timestamp of genesis block
	GenesisTimestamp = int64(0)
	// GenesisCoinbase coinbase address of genesis block
	GenesisCoinbase = common.HexToAddress("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
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

// NewGenesisBlock generates genesis block
func NewGenesisBlock(conf *corepb.Genesis, consensus Consensus, sto storage.Storage) (*Block, error) {
	if conf == nil {
		return nil, ErrNilArgument
	}
	blockState, err := NewBlockState(consensus, sto)
	if err != nil {
		return nil, err
	}
	genesisBlock := &Block{
		BlockData: &BlockData{
			BlockHeader: &BlockHeader{
				hash:       GenesisHash,
				parentHash: GenesisHash,
				chainID:    conf.Meta.ChainId,
				coinbase:   GenesisCoinbase,
				timestamp:  GenesisTimestamp,
				alg:        algorithm.SECP256K1,
			},
			transactions: make(Transactions, 0),
			height:       GenesisHeight,
		},
		storage:   sto,
		state:     blockState,
		consensus: consensus,
		sealed:    false,
	}
	if err := genesisBlock.BeginBatch(); err != nil {
		return nil, err
	}
	for i, v := range conf.GetConsensus().GetDpos().GetDynasty() {
		member := common.HexToAddress(v)

		candidatePb := &dpospb.Candidate{
			Address:   member.Bytes(),
			Collatral: make([]byte, 0),
			VotePower: make([]byte, 0),
		}

		candidate, err := proto.Marshal(candidatePb)
		if err != nil {
			return nil, err
		}

		genesisBlock.State().dposState.CandidateState().Put(member.Bytes(), candidate)
		genesisBlock.State().dposState.DynastyState().Put(byteutils.FromInt32(int32(i)), member.Bytes())

	}

	for _, dist := range conf.TokenDistribution {
		addr := common.HexToAddress(dist.Address)
		balance, err := util.NewUint128FromString(dist.Value)
		if err != nil {
			if err := genesisBlock.RollBack(); err != nil {
				return nil, err
			}
			return nil, err
		}

		err = genesisBlock.state.AddBalance(addr, balance)
		if err != nil {
			logging.Console().WithFields(logrus.Fields{
				"err": err,
			}).Info("add balance failed at newGenesis")
			if err := genesisBlock.RollBack(); err != nil {
				return nil, err
			}
			return nil, err
		}
	}

	initialMessage := "Genesis block of MediBloc"

	initialTx, err := NewTransaction(
		conf.Meta.ChainId,
		GenesisCoinbase, GenesisCoinbase,
		util.Uint128Zero(), 1,
		TxPayloadBinaryType,
		[]byte(initialMessage),
	)
	if err != nil {
		return nil, err
	}
	initialTx.SetTimestamp(GenesisTimestamp)

	hash, err := initialTx.CalcHash()
	if err != nil {
		return nil, err
	}
	initialTx.hash = hash
	initialTx.alg = genesisBlock.Alg()

	pbTx, err := initialTx.ToProto()
	if err != nil {
		return nil, err
	}

	txBytes, err := proto.Marshal(pbTx)
	if err != nil {
		return nil, err
	}

	genesisBlock.transactions = append(genesisBlock.transactions, initialTx)
	if err := genesisBlock.state.PutTx(initialTx.hash, txBytes); err != nil {
		return nil, err
	}

	if err := genesisBlock.Commit(); err != nil {
		return nil, err
	}

	genesisBlock.accsRoot = genesisBlock.state.AccountsRoot()
	genesisBlock.txsRoot = genesisBlock.state.TransactionsRoot()
	genesisBlock.certificationRoot = genesisBlock.state.CertificationRoot()
	genesisBlock.dposRoot, err = genesisBlock.state.DposState().RootBytes()
	if err != nil {
		return nil, err
	}

	genesisBlock.sealed = true

	return genesisBlock, nil
}

// CheckGenesisBlock checks if a block is genesis block
func CheckGenesisBlock(block *Block) bool {
	if block == nil {
		return false
	}
	if byteutils.Equal(block.Hash(), GenesisHash) {
		return true
	}
	return false
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

	accounts, err := block.State().accState.AccountState().Accounts()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to get accounts from block.")
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

	for _, account := range accounts {
		contains := false
		for _, token := range tokenDist {
			if token.Address == byteutils.Bytes2Hex(account.Address()) {
				balance, err := util.NewUint128FromString(token.Value)
				if err != nil {
					logging.Console().WithFields(logrus.Fields{
						"err": err,
					}).Error("Failed to convert balance from string to uint128.")
					return false
				}
				if balance.Cmp(account.Balance()) != 0 {
					logging.Console().WithFields(logrus.Fields{
						"balanceInBlock": account.Balance(),
						"balanceInConf":  balance,
						"account":        byteutils.Bytes2Hex(account.Address()),
					}).Error("Genesis's token balance does not match.")
					return false
				}
				contains = true
				break
			}
		}
		if !contains {
			logging.Console().WithFields(logrus.Fields{
				"account": byteutils.Bytes2Hex(account.Address()),
			}).Error("Accounts of token distribution don't match.")
			return false
		}
	}

	return true
}
