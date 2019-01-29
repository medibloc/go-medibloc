package blockutil

import (
	"testing"

	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/genesisutil"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/stretchr/testify/require"
)

// NewTestGenesisConf returns a genesis configuration for tests.
func NewTestGenesisConf(t *testing.T, dynastySize int) (genesis *corepb.Genesis, dynasties keyutil.AddrKeyPairs, distributed keyutil.AddrKeyPairs) {
	param := genesisutil.DefaultConfigParam()
	param.ChainID = ChainID
	param.DynastySize = dynastySize
	conf := genesisutil.ConfigGenerator(param)
	genesis, err := genesisutil.ConvertGenesisConf(conf)
	require.NoError(t, err)

	for i, s := range conf.Secrets {
		addr, err := common.HexToAddress(s.Public)
		require.NoError(t, err)
		key := conf.Secrets.Key(s.Public)
		pair := &keyutil.AddrKeyPair{Addr: addr, PrivKey: key}

		if i < dynastySize {
			dynasties = append(dynasties, pair)
		}
		distributed = append(distributed, pair)
	}

	return genesis, dynasties, distributed
}

// NewTestGenesisBlock returns a genesis block for tests.
func NewTestGenesisBlock(t *testing.T, dynastySize int) (genesis *core.Block, dynasties keyutil.AddrKeyPairs, distributed keyutil.AddrKeyPairs) {
	conf, dynasties, distributed := NewTestGenesisConf(t, dynastySize)
	s, err := storage.NewMemoryStorage()
	require.NoError(t, err)
	d := dpos.New(dynastySize)
	genesis, err = core.NewGenesisBlock(conf, d, s)
	require.NoError(t, err)

	return genesis, dynasties, distributed
}
