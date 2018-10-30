package util_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/util"
	"github.com/stretchr/testify/require"
)

func TestUint128_MulWithFloat(t *testing.T) {
	t.Log(util.NewUint128FromUint(40).MulWithFloat("0.3"))
	t.Log(util.NewUint128FromUint(4).MulWithFloat("0.3333333333333333"))
	t.Log(util.NewUint128FromUint(3).MulWithFloat("0.5"))
	t.Log(util.NewUint128FromUint(300000000000).MulWithFloat("0.100000001"))

	u, err := util.NewUint128FromString("10000000000000000000000")
	require.NoError(t, err)
	t.Log(u.MulWithFloat(core.InflationRate))

}
