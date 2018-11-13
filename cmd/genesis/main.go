package main

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"io/ioutil"
)

func main() {
	bytes, err := ioutil.ReadFile("conf/testnet/genesis.conf")
	if err != nil {
		fmt.Println("failed to load genesis conf file")
		return
	}
	pbGenesis:= new(corepb.Genesis)
	if err := proto.UnmarshalText(string(bytes), pbGenesis); err != nil {
		fmt.Println("failed unmarshal to corepb.Genesis")
		return
	}

	hash, err := CalcGenesisHashFromGenesisConfig(pbGenesis)
	if err != nil {
		fmt.Println("failed to calculate genesis hash")
		return
	}

	fmt.Println("genesis hash:", byteutils.Bytes2Hex(hash))
	return

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
