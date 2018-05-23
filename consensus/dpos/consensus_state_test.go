package dpos_test

import (
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/util/test"
	"github.com/stretchr/testify/assert"
)

func TestLoadConsensusState(t *testing.T) {
	genesis, _, _ := test.NewTestGenesisBlock(t)
	cs, err := dpos.NewConsensusState(nil, genesis.Storage())
	assert.NoError(t, err)

	time.Sleep(10)

	root1, err := cs.RootBytes()
	newCs, err := dpos.LoadConsensusState(root1, genesis.Storage())
	assert.NoError(t, err)
	root2, err := newCs.RootBytes()
	assert.NoError(t, err)
	assert.Equal(t, root1, root2)
}

func TestClone(t *testing.T) {
	genesis, _, _ := test.NewTestGenesisBlock(t)
	cs, err := dpos.NewConsensusState(nil, genesis.Storage())
	assert.NoError(t, err)

	clone, err := cs.Clone()
	assert.NoError(t, err)
	assert.Equal(t, cs, clone)
}
