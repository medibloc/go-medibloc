package genesisutil

import (
	"os"
	"strconv"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/crypto"
	"github.com/medibloc/go-medibloc/crypto/signature/algorithm"
	"github.com/medibloc/go-medibloc/util/byteutils"
)

// Genesis util's default configuration.
const (
	DynastySize    = 21
	ChainID        = 181228
	TokenDist      = 25
	TotalTokens    = 5000000000
	Stake          = 100000000
	Collateral     = 1000000
	GenesisMessage = "Genesis block of MediBloc"
)

// DefaultConfig returns default config.
func DefaultConfig() *Config {
	return GenerateGenesisConfig(DefaultConfigParam())
}

// DefaultConfigParam returns default config param.
func DefaultConfigParam() *GenerateGenesisConfigParam {
	return &GenerateGenesisConfigParam{
		ChainID:        ChainID,
		DynastySize:    DynastySize,
		TokenDist:      TokenDist,
		TotalTokens:    TotalTokens,
		GenesisMessage: GenesisMessage,
		Stake:          Stake,
		Collateral:     Collateral,
	}
}

// GenerateGenesisConfigParam represents parameters for generating genesis configuration.
type GenerateGenesisConfigParam struct {
	ChainID        uint32
	DynastySize    int
	TokenDist      int
	TotalTokens    int64
	GenesisMessage string
	Stake          int64
	Collateral     int64
}

// GenerateGenesisConfig generates genesis configuration.
func GenerateGenesisConfig(param *GenerateGenesisConfigParam) *Config {
	secrets := secretn(param.TokenDist)
	return GenerateGenesisConfigWithSecret(param, secrets)
}

// GenerateGenesisConfigWithSecret generates genesis configuration with given secrets.
func GenerateGenesisConfigWithSecret(param *GenerateGenesisConfigParam, secrets Secrets) *Config {
	meta := &Meta{
		ChainID:     param.ChainID,
		DynastySize: param.DynastySize,
	}
	txs := generateTransactions(param, secrets)

	return &Config{
		Meta:        meta,
		Secrets:     secrets,
		Transaction: txs,
	}
}

func generateTransactions(param *GenerateGenesisConfigParam, secrets []*Secret) []*Transaction {
	transactions := make([]*Transaction, 0, 1024)
	n := int64(len(secrets))

	// Genesis Transaction
	transactions = append(transactions, &Transaction{
		Type: transaction.TxOpGenesis,
		From: core.GenesisCoinbase.Hex(),
		Data: param.GenesisMessage,
	})

	// Genesis Distribution
	for i, secret := range secrets {
		value := param.TotalTokens / n
		if i < int(param.TotalTokens%n) {
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
	for i := 0; i < param.DynastySize; i++ {
		secret := secrets[i]

		// Staking
		tx := &Transaction{
			Type:  transaction.TxOpStake,
			From:  secret.Public,
			Value: param.Stake,
		}
		transactions = append(transactions, tx)

		// Register alias
		tx = &Transaction{
			Type:  transaction.TxOpRegisterAlias,
			From:  secret.Public,
			Data:  "blockproducer" + strconv.Itoa(i),
			Value: param.Collateral,
		}
		transactions = append(transactions, tx)

		// Become candidate
		tx = &Transaction{
			Type:  transaction.TxOpBecomeCandidate,
			From:  secret.Public,
			Data:  "medibloc.org",
			Value: param.Collateral,
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

func secretn(n int) []*Secret {
	secrets := make([]*Secret, 0, n)
	for i := 0; i < n; i++ {
		secrets = append(secrets, secret())
	}
	return secrets
}

func secret() *Secret {
	public, private := keypair()
	return &Secret{
		Public:  public,
		Private: private,
	}
}

func keypair() (public string, private string) {
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
