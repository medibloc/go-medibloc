package blockutil_test

import (
	"testing"
	"time"

	"github.com/medibloc/go-medibloc/consensus/dpos"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
)

func TestBlockBuilder_ChildNextDynasty(t *testing.T) {
	bb := blockutil.New(t, blockutil.DynastySize).Genesis()
	dynastyInterval := dpos.New(blockutil.DynastySize).DynastyInterval()
	t.Log("Dynasty Interval:", dynastyInterval)

	bb = bb.ChildNextDynasty().SignProposer()
	b := bb.Build()
	t.Log("Block timestamp of next dynasty:", b.Timestamp())
	assert.Equal(t, int64(dynastyInterval/time.Second), b.Timestamp())

	bb = bb.ChildNextDynasty().SignProposer()
	b = bb.Build()
	t.Log("Block timestamp of next dynasty:", b.Timestamp())
	assert.Equal(t, int64(dynastyInterval/time.Second)*2, b.Timestamp())

	bb = bb.Child().SignProposer().ChildNextDynasty().SignProposer()
	b = bb.Build()
	t.Log("Block timestamp of next dynasty:", b.Timestamp())
	assert.Equal(t, int64(dynastyInterval/time.Second)*3, b.Timestamp())

}
