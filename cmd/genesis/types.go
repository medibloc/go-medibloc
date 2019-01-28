package main

import (
	"github.com/medibloc/go-medibloc/crypto/signature"
	"github.com/medibloc/go-medibloc/crypto/signature/secp256k1"
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
	Secrets     Secrets
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

type Secrets []*Secret

func (ss Secrets) key(from string) signature.PrivateKey {
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
