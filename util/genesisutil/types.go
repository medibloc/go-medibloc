package genesisutil

import (
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Meta        *Meta
	Secrets     Secrets
	Transaction []*Transaction
}

func BytesToConfig(buf []byte) (*Config, error) {
	var conf *Config
	if err := yaml.Unmarshal(buf, &conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func ConfigToBytes(conf *Config) ([]byte, error) {
	out, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type Meta struct {
	ChainID     uint32
	DynastySize int
}

type Secret struct {
	Public  string
	Private string
}

type Secrets []*Secret

func (ss Secrets) Key(from string) signature.PrivateKey {
	for _, s := range ss {
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

type Transaction struct {
	Type  string
	From  string
	To    string
	Data  string
	Value int64
}
