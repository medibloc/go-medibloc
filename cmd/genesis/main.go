package main

import (
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/storage"
)

func main() {

	//aa := corepb.Genesis{
	//	Meta:              nil,
	//	TokenDistribution: nil,
	//	Transactions:      nil,
	//}

}

// CalcGenesisHashFromGenesisConfig returns hash of genesis block
func CalcGenesisHashFromGenesisConfig(genesis *corepb.Genesis) ([]byte, error) {
	consenesus := dpos.New(int(genesis.Meta.DynastySize))
	stor, err := storage.NewMemoryStorage()
	if err != nil {
		return nil, err
	}

	b, err := core.NewGenesisBlock(genesis, consenesus, medlet.DefaultTxMap, stor)
	return b.Hash(), nil
}
