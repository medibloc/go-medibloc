package dpos_test

import (
	"testing"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/stretchr/testify/require"
)

func TestMakeNewDynasty(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()

	cs := bb.Build().State().DposState().CandidateState()
	ds := bb.Build().State().DposState().DynastyState()

	genesisDynasty, err := dpos.DynastyStateToDynasty(ds)
	require.NoError(t, err)
	t.Log("genesis dynasty", genesisDynasty)

	candidates, err := dpos.SortByVotePower(cs)
	require.NoError(t, err)

	newDynasty := dpos.MakeNewDynasty(candidates, testutil.DynastySize)
	t.Log("new dynasty:", newDynasty)
}
