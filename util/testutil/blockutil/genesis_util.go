package blockutil

import (
	"strconv"
	"testing"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/core"
	corepb "github.com/medibloc/go-medibloc/core/pb"
	corestate "github.com/medibloc/go-medibloc/core/state"
	"github.com/medibloc/go-medibloc/core/transaction"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util"
	"github.com/medibloc/go-medibloc/util/testutil/keyutil"
	"github.com/stretchr/testify/require"
)

// NewTestGenesisConf returns a genesis configuration for tests.
func NewTestGenesisConf(t *testing.T, dynastySize int) (conf *corepb.Genesis, dynasties keyutil.AddrKeyPairs, distributed keyutil.AddrKeyPairs) {
	conf = &corepb.Genesis{
		Meta: &corepb.GenesisMeta{
			ChainId:     ChainID,
			DynastySize: uint32(dynastySize),
		},
		TokenDistribution: nil,
		Transactions:      nil,
	}

	var dynasty []string
	var tokenDist []*corepb.GenesisTokenDistribution
	txs := make([]*corestate.Transaction, 0)

	builder := New(t, dynastySize).Tx().
		ChainID(ChainID)

	for i := 0; i < dynastySize; i++ {
		keypair := keyutil.NewAddrKeyPair(t)
		dynasty = append(dynasty, keypair.Addr.Hex())
		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: keypair.Addr.Hex(),
			Balance: "400000000000000000000",
		})

		dynasties = append(dynasties, keypair)
		distributed = append(distributed, keypair)

		staking, err := util.NewUint128FromString("100000000000000000000")
		require.NoError(t, err)
		tx := builder.
			Type(transaction.TxOpStake).
			ValueRaw(staking).
			Nonce(1).
			SignKey(keypair.PrivKey).
			Build()
		txs = append(txs, tx)

		collateral, err := util.NewUint128FromString("1000000000000000000")
		require.NoError(t, err)
		aliasPayload := &transaction.RegisterAliasPayload{AliasName: "accountalias" + strconv.Itoa(i)}
		tx = builder.
			Type(transaction.TxOpRegisterAlias).
			ValueRaw(collateral).
			Nonce(2).
			Payload(aliasPayload).
			SignKey(keypair.PrivKey).
			Build()
		txs = append(txs, tx)

		tx = builder.
			Type(transaction.TxOpBecomeCandidate).
			ValueRaw(collateral).
			Nonce(3).
			SignKey(keypair.PrivKey).
			Build()
		txs = append(txs, tx)

		candidateID := tx.Hash()
		votePayload := &transaction.VotePayload{
			CandidateIDs: [][]byte{candidateID},
		}
		tx = builder.
			Type(transaction.TxOpVote).
			Nonce(4).
			Payload(votePayload).
			SignKey(keypair.PrivKey).
			Build()
		txs = append(txs, tx)
	}

	distCnt := 40 - dynastySize

	for i := 0; i < distCnt; i++ {
		keypair := keyutil.NewAddrKeyPair(t)
		tokenDist = append(tokenDist, &corepb.GenesisTokenDistribution{
			Address: keypair.Addr.Hex(),
			Balance: "400000000000000000000",
		})
		distributed = append(distributed, keypair)
	}

	conf.TokenDistribution = tokenDist
	conf.Transactions = make([]*corepb.Transaction, 0)
	for _, v := range txs {
		pbMsg, err := v.ToProto()
		require.NoError(t, err)
		pbTx, ok := pbMsg.(*corepb.Transaction)
		require.True(t, ok)
		conf.Transactions = append(conf.Transactions, pbTx)
	}

	return conf, dynasties, distributed
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
