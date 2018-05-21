package core

import (
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

var (
	// GenesisHash is hash of genesis block
	GenesisHash = common.Hash{}
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
	var members []*common.Address
	for _, v := range conf.GetConsensus().GetDpos().GetDynasty() {
		member := common.HexToAddress(v)
		if err := genesisBlock.State().AddCandidate(member, util.Uint128Zero()); err != nil {
			return nil, err
		}
		members = append(members, &member)
	}
	genesisBlock.State().SetDynasty(members, int64(0))

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
	if block.Hash().Equals(GenesisHash) {
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
