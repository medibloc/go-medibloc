package main

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	dposState "github.com/medibloc/go-medibloc/consensus/dpos/state"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	coreState "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

func randomPrivateKeys() []string {
	privKeys := make([]string, 25)
	for i := 0; i < 25; i++ {
		key, err := crypto.GenerateKey(algorithm.SECP256K1)
		if err != nil {
			panic("failed to generate key")
		}
		keyBytes, err := key.Encoded()
		if err != nil {
			panic("failed to generate key")
		}
		privKeys[i] = byteutils.Bytes2Hex(keyBytes)
	}
	return privKeys
}

func main() {
	// privKeys := testnet_privKeys()
	privKeys := randomPrivateKeys()

	const chainID = 181228
	const dynastySize = 21

	chainCfg := new(medletpb.ChainConfig)
	chainCfg.Proposers = make([]*medletpb.ProposerConfig, dynastySize)

	conf := &corepb.Genesis{
		Meta: &corepb.GenesisMeta{
			ChainId:     chainID,
			DynastySize: uint32(dynastySize),
		},
		TokenDistribution: nil,
		Transactions:      nil,
	}

	var tokenDist []*corepb.GenesisTokenDistribution
	txs := make([]*coreState.Transaction, 0)

	for i, v := range privKeys[:dynastySize] {
		key, err := secp256k1.NewPrivateKeyFromHex(v)
		if err != nil {
			fmt.Println("failed to convert string to private key")
			return
		}
		sig, _ := crypto.NewSignature(algorithm.SECP256K1)
		sig.InitSign(key)

		addr, err := common.PublicKeyToAddress(key.PublicKey())

		proposer := &medletpb.ProposerConfig{
			Proposer: addr.Hex(),
			Privkey:  v,
			Coinbase: addr.Hex(),
		}

		chainCfg.Proposers[i] = proposer

		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: addr.Hex(),
			Balance: "320000000000000000000",
		})

		vesting, _ := util.NewUint128FromString("100000000000000000000")
		collateral, _ := util.NewUint128FromString(transaction.CandidateCollateralMinimum)
		aliasCollateral, _ := util.NewUint128FromString(transaction.AliasCollateralMinimum)

		tx := new(coreState.Transaction)

		tx.SetChainID(chainID)
		tx.SetValue(util.NewUint128())

		txVest, _ := tx.Clone()
		txVest.SetTxType(coreState.TxOpStake)
		txVest.SetValue(vesting)
		txVest.SetNonce(1)
		txVest.SignThis(key)

		aliasPayload := &transaction.RegisterAliasPayload{AliasName: "testnetbpname" + strconv.Itoa(i)}
		aliasePayloadBytes, _ := aliasPayload.ToBytes()

		txAlias, err := tx.Clone()
		txAlias.SetTxType(coreState.TxOpRegisterAlias)
		txAlias.SetValue(aliasCollateral)
		txAlias.SetNonce(2)
		txAlias.SetPayload(aliasePayloadBytes)
		txAlias.SignThis(key)

		becomeCandidatePayload := &transaction.BecomeCandidatePayload{URL: "www.medibloc.org"}
		becomeCandidatePayloadBytes, _ := becomeCandidatePayload.ToBytes()

		txCandidate, _ := tx.Clone()
		txCandidate.SetTxType(dposState.TxOpBecomeCandidate)
		txCandidate.SetValue(collateral)
		txCandidate.SetNonce(3)
		txCandidate.SetPayload(becomeCandidatePayloadBytes)
		txCandidate.SignThis(key)

		votePayload := new(transaction.VotePayload)
		candidateIds := make([][]byte, 0)
		votePayload.CandidateIDs = append(candidateIds, txCandidate.Hash())
		votePayloadBytes, _ := votePayload.ToBytes()

		txVote, _ := tx.Clone()
		txVote.SetTxType(dposState.TxOpVote)
		txVote.SetNonce(4)
		txVote.SetPayload(votePayloadBytes)
		txVote.SignThis(key)

		txs = append(txs, txVest, txAlias, txCandidate, txVote)
	}

	for _, v := range privKeys[dynastySize:] {
		key, err := secp256k1.NewPrivateKeyFromHex(v)
		if err != nil {
			fmt.Println("failed to convert string to private key")
			return
		}
		sig, _ := crypto.NewSignature(algorithm.SECP256K1)
		sig.InitSign(key)

		addr, err := common.PublicKeyToAddress(key.PublicKey())

		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: addr.Hex(),
			Balance: "320000000000000000000",
		})
	}

	conf.TokenDistribution = tokenDist
	conf.Transactions = make([]*corepb.Transaction, 0)
	for _, v := range txs {
		pbMsg, _ := v.ToProto()
		pbTx, _ := pbMsg.(*corepb.Transaction)
		conf.Transactions = append(conf.Transactions, pbTx)
	}

	chainCfgTxt := proto.MarshalTextString(chainCfg)
	err := ioutil.WriteFile("./cmd/genesis/node.conf", []byte(chainCfgTxt), 400)
	if err != nil {
		panic("failed to save file chain config")
	}

	genesisTxt := proto.MarshalTextString(conf)
	err = ioutil.WriteFile("./cmd/genesis/genesis.conf", []byte(genesisTxt), 400)
	if err != nil {
		panic("failed to save file genesis config")
	}
}
