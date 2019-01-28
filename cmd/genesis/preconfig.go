package main

import (
	"os"
	"strconv"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util/byteutils"
	yaml "gopkg.in/yaml.v2"
)

func preConfig() []byte {
	config := &Config{
		ChainID:     ChainID,
		DynastySize: DynastySize,
	}
	secrets := secretn(TokenDist)
	txs := transactions(TotalTokens, DynastySize, secrets)

	cont := &Container{
		Config:      config,
		Secrets:     secrets,
		Transaction: txs,
	}

	out, err := yaml.Marshal(cont)
	if err != nil {
		panic(err)
	}

	return out
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
		From: core.GenesisCoinbase.Hex(),
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
			From:  core.GenesisCoinbase.Hex(),
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
			Data:  "blockproducer" + strconv.Itoa(i),
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
