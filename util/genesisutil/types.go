package genesisutil

import (
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	yaml "gopkg.in/yaml.v2"
)

// Config represents genesis configuration.
type Config struct {
	Meta        *Meta
	Secrets     Secrets
	Transaction []*Transaction
}

// BytesToConfig converts bytes slice to Config structure.
func BytesToConfig(buf []byte) (*Config, error) {
	var conf *Config
	if err := yaml.Unmarshal(buf, &conf); err != nil {
		return nil, err
	}
	return conf, nil
}

// ConfigToBytes converts Config to bytes slice.
func ConfigToBytes(conf *Config) ([]byte, error) {
	out, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Meta has chain id and dynasty size settings.
type Meta struct {
	ChainID     uint32
	DynastySize int
}

// Secret represents public address and private key pair in hex string.
type Secret struct {
	Public  string
	Private string
}

// Key returns private key.
func (s *Secret) Key() signature.PrivateKey {
	key, err := secp256k1.NewPrivateKeyFromHex(s.Private)
	if err != nil {
		panic(err)
	}
	return key
}

// Secrets is an array list of Secret.
type Secrets []*Secret

// Key returns private key of given public address.
func (ss Secrets) Key(from string) signature.PrivateKey {
	for _, s := range ss {
		if s.Public == from {
			return s.Key()
		}
	}
	return nil
}

// Transaction represents transaction data in genesis configuration.
type Transaction struct {
	Type  string
	From  string
	To    string
	Data  string
	Value int64
}
