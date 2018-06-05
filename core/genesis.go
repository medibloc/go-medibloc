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
			header: &BlockHeader{
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
		storage: sto,
		state:   blockState,
		sealed:  false,
	}
	if err := genesisBlock.BeginBatch(); err != nil {
		return nil, err
	}
	dynastySize := int(conf.GetMeta().DynastySize)
	var members []*common.Address
	for _, v := range conf.GetConsensus().GetDpos().GetDynasty() {
		member := common.HexToAddress(v)
		if err := genesisBlock.State().AddCandidate(member, util.Uint128Zero()); err != nil {
			return nil, err
		}
		members = append(members, &member)
	}
	genesisBlock.State().SetDynasty(members, dynastySize, int64(0))

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

	hash, err := initialTx.calcHash()
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

	genesisBlock.header.accsRoot = genesisBlock.state.AccountsRoot()
	genesisBlock.header.txsRoot = genesisBlock.state.TransactionsRoot()
	genesisBlock.header.candidacyRoot = genesisBlock.state.CandidacyRoot()
	genesisBlock.header.consensusRoot, err = genesisBlock.state.ConsensusRoot()
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

	if block.State().DynastySize() != int(genesis.Meta.DynastySize) {
		logging.Console().WithFields(logrus.Fields{
			"sizeInBlock":  block.State().DynastySize(),
			"sizeInConfig": genesis.Meta.DynastySize,
		}).Error("Genesis's dynasty size does not match.")
		return false
	}

	members, err := block.State().Dynasty()
	if err != nil {
		logging.Console().WithFields(logrus.Fields{
			"block":   block,
			"genesis": genesis,
			"err":     err,
		}).Error("Failed to get dynasties.")
		return false
	}
	if len(members) != len(genesis.GetConsensus().GetDpos().GetDynasty()) {
		logging.Console().WithFields(logrus.Fields{
			"block":   block,
			"genesis": genesis,
			"members": members,
		}).Error("Size of genesis dynasties does not match.")
		return false
	}
	for _, member := range members {
		contains := false
		for _, mm := range genesis.GetConsensus().GetDpos().GetDynasty() {
			if member.Equals(common.HexToAddress(mm)) {
				contains = true
				break
			}
		}
		if !contains {
			logging.Console().WithFields(logrus.Fields{
				"member": member,
			}).Error("Members of genesis don't match.")
			return false
		}
	}

	return true
}
