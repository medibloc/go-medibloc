package core

import (
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
)

var (
	// GenesisHash is hash of genesis block
	GenesisHash = common.BytesToHash(make([]byte, common.HashLength))
	// GenesisTimestamp is timestamp of genesis block
	GenesisTimestamp = int64(0)
	// GenesisCoinbase coinbase address of genesis block
	GenesisCoinbase = common.HexToAddress("ff7b1d22d234bde673bfa783d6c0c6b835aab407")
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
func NewGenesisBlock(conf *corepb.Genesis, storagePath string) (*Block, error) {
	if conf == nil {
		return nil, ErrNilArgument
	}

	sto, err := storage.NewStorage(storagePath)
	if err != nil {
		return nil, err
	}

	blockState, err := NewBlockState(sto)
	if err != nil {
		return nil, err
	}

	genesisBlock := &Block{
		header: &BlockHeader{
			hash:       GenesisHash,
			parentHash: GenesisHash,
			chainID:    conf.Meta.ChainId,
			coinbase:   GenesisCoinbase,
			timestamp:  GenesisTimestamp,
			alg:        algorithm.SECP256K1,
		},
		transactions: make(Transactions, 0),
		storage:      sto,
		state:        blockState,
		height:       1,
		sealed:       false,
	}

	if err := genesisBlock.BeginBatch(); err != nil {
		return nil, err
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
