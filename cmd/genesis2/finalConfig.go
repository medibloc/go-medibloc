package main

import (
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/core/txfactory"
	"github.com/medibloc/go-medibloc/util"
	yaml "gopkg.in/yaml.v2"
)

func finalConfig(buf []byte) (genesisBuf []byte, proposerBuf []byte) {
	var cont Container
	err := yaml.Unmarshal(buf, &cont)
	if err != nil {
		panic(err)
	}
	genesis := &corepb.Genesis{
		Meta: &corepb.GenesisMeta{
			ChainId:     cont.Config.ChainID,
			DynastySize: cont.Config.DynastySize,
		},
	}

	nonce := make(map[string]uint64)
	candidateTxHash := make(map[string][]byte)

	factory := txfactory.New(cont.Config.ChainID)

	txs := make([]*corepb.Transaction, 0)
	for _, tx := range cont.Transaction {
		nnonce := nonce[tx.From] + 1
		nonce[tx.From] = nnonce

		var ttx *core.Transaction
		switch tx.Type {
		case transaction.TxOpGenesis:
			ttx, err = factory.Genesis(tx.Data)
		case transaction.TxOpGenesisDistribution:
			ttx, err = factory.GenesisDistribution(convAddr(tx.To), convValue(tx.Value), nnonce)
		case transaction.TxOpStake:
			ttx, err = factory.Stake(cont.Secrets.key(tx.From), convValue(tx.Value), nnonce)
		case transaction.TxOpRegisterAlias:
			ttx, err = factory.RegisterAlias(cont.Secrets.key(tx.From), convValue(tx.Value), tx.Data, nnonce)
		case transaction.TxOpBecomeCandidate:
			ttx, err = factory.BecomeCandidate(cont.Secrets.key(tx.From), convValue(tx.Value), tx.Data, nnonce)
			candidateTxHash[tx.From] = ttx.Hash()
		case transaction.TxOpVote:
			cid, exist := candidateTxHash[tx.From]
			if !exist {
				panic("candidate not exist")
			}
			ttx, err = factory.Vote(cont.Secrets.key(tx.From), [][]byte{cid}, nnonce)
		}
		if err != nil {
			panic(err)
		}

		msg, err := ttx.ToProto()
		if err != nil {
			panic(err)
		}
		txs = append(txs, msg.(*corepb.Transaction))
	}
	genesis.Transactions = txs
	genesisBuf = []byte(proto.MarshalTextString(genesis))

	proposerBuf = ProposerOutput(cont)

	return genesisBuf, proposerBuf
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
