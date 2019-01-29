package genesisutil

import (
	"errors"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/core/txfactory"
	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util"
)

func ConvertProposerConfBytes(conf *Config) []byte {
	return []byte(proto.MarshalTextString(ConvertProposerConf(conf)))
}

func ConvertProposerConf(conf *Config) *medletpb.Config {
	proposers := make([]*medletpb.ProposerConfig, 0, len(conf.Secrets))
	for i, s := range conf.Secrets {
		if i >= int(conf.Meta.DynastySize) {
			break
		}
		proposers = append(proposers, &medletpb.ProposerConfig{
			Proposer: s.Public,
			Privkey:  s.Private,
			Coinbase: s.Public,
		})
	}

	return &medletpb.Config{
		Chain: &medletpb.ChainConfig{
			Proposers: proposers,
		},
	}
}

func ConvertGenesisConfBytes(conf *Config) ([]byte, error) {
	genesis, err := ConvertGenesisConf(conf)
	if err != nil {
		return nil, err
	}
	return []byte(proto.MarshalTextString(genesis)), nil
}

func ConvertGenesisConf(conf *Config) (*corepb.Genesis, error) {
	genesis := &corepb.Genesis{
		Meta: &corepb.GenesisMeta{
			ChainId:     conf.Meta.ChainID,
			DynastySize: uint32(conf.Meta.DynastySize),
		},
	}

	nonce := make(map[string]uint64)
	candidateTxHash := make(map[string][]byte)

	factory := txfactory.New(conf.Meta.ChainID)

	txs := make([]*corepb.Transaction, 0)
	for _, tx := range conf.Transaction {
		nnonce := nonce[tx.From] + 1
		nonce[tx.From] = nnonce

		var (
			ttx *core.Transaction
			err error
		)
		switch tx.Type {
		case transaction.TxOpGenesis:
			ttx, err = factory.Genesis(tx.Data)
		case transaction.TxOpGenesisDistribution:
			ttx, err = factory.GenesisDistribution(convAddr(tx.To), convValue(tx.Value), nnonce)
		case transaction.TxOpStake:
			ttx, err = factory.Stake(conf.Secrets.Key(tx.From), convValue(tx.Value), nnonce)
		case transaction.TxOpRegisterAlias:
			ttx, err = factory.RegisterAlias(conf.Secrets.Key(tx.From), convValue(tx.Value), tx.Data, nnonce)
		case transaction.TxOpBecomeCandidate:
			ttx, err = factory.BecomeCandidate(conf.Secrets.Key(tx.From), convValue(tx.Value), tx.Data, nnonce)
			candidateTxHash[tx.From] = ttx.Hash()
		case transaction.TxOpVote:
			cid, exist := candidateTxHash[tx.From]
			if !exist {
				return nil, errors.New("candidate not exist")
			}
			ttx, err = factory.Vote(conf.Secrets.Key(tx.From), [][]byte{cid}, nnonce)
		}
		if err != nil {
			return nil, err
		}

		msg, err := ttx.ToProto()
		if err != nil {
			return nil, err
		}
		txs = append(txs, msg.(*corepb.Transaction))
	}
	genesis.Transactions = txs
	return genesis, nil
}

func convValue(v int64) *util.Uint128 {
	value := util.NewUint128FromUint(uint64(v))
	value, err := value.Mul(util.NewUint128FromUint(1000000000000))
	if err != nil {
		panic(err)
	}
	return value
}

func convAddr(addrStr string) common.Address {
	addr, err := common.HexToAddress(addrStr)
	if err != nil {
		panic(err)
	}
	return addr
}
