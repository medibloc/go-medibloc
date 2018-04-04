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
  GenesisHash      = common.BytesToHash(make([]byte, common.HashLength))
  GenesisTimestamp = int64(0)
  GenesisCoinbase  = common.HexToAddress("ff7b1d22d234bde673bfa783d6c0c6b835aab407")
)

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

// TEST VERSION
func NewGenesisBlock(conf *corepb.Genesis, storagePath string) (*Block, error) {
  if conf == nil {
    return nil, ErrNilArgument
  }

  sto, err := storage.NewStorage(storagePath)
  if err != nil {
    return nil, err
  }

  accState, err := NewAccountStateBatch(nil, sto)
  if err != nil {
    return nil, err
  }

  txsState, err := NewTrieBatch(nil, sto)
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
    accState:     accState,
    txsState:     txsState,
    height:       1,
    sealed:       false,
  }

  if err := genesisBlock.Begin(); err != nil {
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

    err = genesisBlock.accState.AddBalance(addr.Bytes(), balance)
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
  if err := genesisBlock.txsState.Put(initialTx.hash.Bytes(), txBytes); err != nil {
    return nil, err
  }

  if err := genesisBlock.Commit(); err != nil {
    return nil, err
  }

  genesisBlock.header.accsRoot = common.BytesToHash(genesisBlock.accState.RootHash())
  genesisBlock.header.txsRoot = common.BytesToHash(genesisBlock.txsState.RootHash())

  genesisBlock.sealed = true

  return genesisBlock, nil
}

func CheckGenesisBlock(block *Block) bool {
  if block == nil {
    return false
  }
  if block.Hash().Equals(GenesisHash) {
    return true
  }
  return false
}
