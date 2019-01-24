package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/byteutils"
	yaml "gopkg.in/yaml.v2"
)

const (
	DynastySize    = 21
	ChainID        = 181228
	TokenDist      = 25
	TotalTokens    = 5000000000
	Stake          = 100000000
	Collateral     = 1000000
	GenesisMessage = "Genesis block of MediBloc"
)

type Container struct {
	Config      *Config
	Secret      []*Secret
	Transaction []*Transaction
}

type Config struct {
	ChainID     uint32
	DynastySize uint32
}

type Secret struct {
	Public  string
	Private string
}

type Transaction struct {
	Type  string
	From  string
	To    string
	Data  string
	Value int64
}

func printUsage() {
	fmt.Println("generate pre <output file>: generate pre configuration")
	fmt.Println("generate final <pre config file> : generate genesis configuration")
}

func main() {
	if len(os.Args) != 3 {
		printUsage()
		os.Exit(1)
	}

	if os.Args[1] == "pre" {
		preConfig(os.Args[2])
		return
	} else if os.Args[1] == "final" {
		buf, err := ioutil.ReadFile(os.Args[2])
		if err != nil {
			printUsage()
			os.Exit(1)
		}
		finalConfig(buf)
		return
	}
	printUsage()
	os.Exit(1)
}

func finalConfig(buf []byte) {
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

	txs := make([]*corepb.Transaction, 0)
	for _, tx := range cont.Transaction {
		nnonce := nonce[tx.From] + 1
		nonce[tx.From] = nnonce

		from, err := common.HexToAddress(tx.From)
		if err != nil {
			panic(err)
		}

		to, err := common.HexToAddress(tx.To)
		if err != nil {
			panic(err)
		}

		value := util.NewUint128FromUint(uint64(tx.Value))
		value, err = value.Mul(util.NewUint128FromUint(1000000000000))
		if err != nil {
			panic(err)
		}

		payload := payloadFromData(tx, candidateTxHash)

		ttx := &corestate.Transaction{}
		ttx.SetTxType(tx.Type)
		ttx.SetFrom(from)
		ttx.SetTo(to)
		ttx.SetValue(value)
		ttx.SetNonce(nnonce)
		ttx.SetChainID(cont.Config.ChainID)
		ttx.SetPayload(payload)
		if tx.From != "" {
			err = ttx.SignThis(key(cont.Secret, tx.From))
			if err != nil {
				panic(err)
			}
		}

		if tx.Type == transaction.TxOpBecomeCandidate {
			candidateTxHash[tx.From] = ttx.Hash()
		}

		msg, err := ttx.ToProto()
		if err != nil {
			panic(err)
		}
		txs = append(txs, msg.(*corepb.Transaction))
	}
	genesis.Transactions = txs
	fmt.Println(proto.MarshalTextString(genesis))
}

func key(secrets []*Secret, from string) signature.PrivateKey {
	for _, s := range secrets {
		if s.Public == from {
			key, err := secp256k1.NewPrivateKeyFromHex(s.Private)
			if err != nil {
				panic(err)
			}
			return key
		}
	}
	return nil
}

func payloadFromData(tx *Transaction, candidateTxHash map[string][]byte) []byte {
	var (
		payload []byte
		err     error
	)
	switch tx.Type {
	case transaction.TxOpGenesis:
		payload, err = (&transaction.DefaultPayload{
			Message: tx.Data,
		}).ToBytes()
	case transaction.TxOpRegisterAlias:
		payload, err = (&transaction.RegisterAliasPayload{
			AliasName: tx.Data,
		}).ToBytes()
	case transaction.TxOpBecomeCandidate:
		payload, err = (&transaction.BecomeCandidatePayload{
			URL: tx.Data,
		}).ToBytes()
	case transaction.TxOpVote:
		hash, exist := candidateTxHash[tx.From]
		if !exist {
			panic("candidate not exist")
		}
		payload, err = (&transaction.VotePayload{
			CandidateIDs: [][]byte{hash},
		}).ToBytes()
	}
	if err != nil {
		panic(err)
	}
	return payload
}

func preConfig(filename string) {
	config := &Config{
		ChainID: ChainID,

		DynastySize: DynastySize,
	}
	secrets := secretn(TokenDist)
	txs := transactions(TotalTokens, DynastySize, secrets)

	cont := &Container{
		Config:      config,
		Secret:      secrets,
		Transaction: txs,
	}

	out, err := yaml.Marshal(cont)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(filename, out, 0600)
	if err != nil {
		panic(err)
	}
}

func secretn(n int) []*Secret {
	secrets := make([]*Secret, 0, n)
	for i := 0; i < n; i++ {
		secrets = append(secrets, secret())
	}
	return secrets
}

func secret() *Secret {
	public, private := generate()
	return &Secret{
		Public:  public,
		Private: private,
	}
}

func generate() (public string, private string) {
	privKey, err := crypto.GenerateKey(algorithm.SECP256K1)
	if err != nil {
		panic(err)
	}
	pub, err := privKey.PublicKey().Compressed()
	if err != nil {
		panic(err)
	}
	pri, err := privKey.Encoded()
	if err != nil {
		os.Exit(1)
	}
	return byteutils.Bytes2Hex(pub), byteutils.Bytes2Hex(pri)
}

func transactions(tokens int64, dynastySize int, secrets []*Secret) []*Transaction {
	transactions := make([]*Transaction, 0, 1024)
	n := int64(len(secrets))

	// Genesis Transaction
	transactions = append(transactions, &Transaction{
		Type: transaction.TxOpGenesis,
		Data: GenesisMessage,
	})

	// Genesis Distribution
	for i, secret := range secrets {
		value := tokens / n
		if i < int(tokens%n) {
			value++
		}

		tx := &Transaction{
			Type:  transaction.TxOpGenesisDistribution,
			To:    secret.Public,
			Value: value,
		}
		transactions = append(transactions, tx)
	}

	// Block Producer
	for i := 0; i < dynastySize; i++ {
		secret := secrets[i]

		// Staking
		tx := &Transaction{
			Type:  transaction.TxOpStake,
			From:  secret.Public,
			Value: Stake,
		}
		transactions = append(transactions, tx)

		// Register alias
		tx = &Transaction{
			Type:  transaction.TxOpRegisterAlias,
			From:  secret.Public,
			Data:  "testnetproducer" + strconv.Itoa(i),
			Value: Collateral,
		}
		transactions = append(transactions, tx)

		// Become candidate
		tx = &Transaction{
			Type:  transaction.TxOpBecomeCandidate,
			From:  secret.Public,
			Data:  "medibloc.org",
			Value: Collateral,
		}
		transactions = append(transactions, tx)

		// Vote
		tx = &Transaction{
			Type: transaction.TxOpVote,
			From: secret.Public,
			Data: secret.Public,
		}
		transactions = append(transactions, tx)
	}

	return transactions
}
