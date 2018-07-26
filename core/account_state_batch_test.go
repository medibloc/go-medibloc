package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/util/testutil"
	"github.com/medibloc/go-medibloc/util/testutil/blockutil"
	"github.com/stretchr/testify/assert"
)

func TestAccountStateBatch_GetAccount(t *testing.T) {
	bb := blockutil.New(t, testutil.DynastySize).Genesis()
	b := bb.Build()

	for i := 0; i < 100; i++ {
		go func() {
			_, err := b.State().GetAccount(bb.TokenDist[0].Addr)
			assert.NoError(t, err)
		}()
	}

}
