// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package core_test

import (
	"testing"

	"github.com/medibloc/go-medibloc/core"
	"github.com/medibloc/go-medibloc/medlet"
	"github.com/stretchr/testify/require"
	"github.com/medibloc/go-medibloc/util/testutil"
	"time"
)

func TestNewBlockChain(t *testing.T) {
	cfg := medlet.DefaultConfig()

	cfg.Chain.BlockCacheSize = 0
	_, err := core.NewBlockChain(cfg)
	require.EqualError(t, err, "Must provide a positive size")
	cfg.Chain.BlockCacheSize = 1

	cfg.Chain.TailCacheSize = 0
	_, err = core.NewBlockChain(cfg)
	require.EqualError(t, err, "Must provide a positive size")
}

func TestRestartNode(t *testing.T) {
	testNet := testutil.NewNetwork(t, testutil.DynastySize)
	defer testNet.Cleanup()

	seed := testNet.NewSeedNode()
	testNet.SetMinerFromDynasties(seed)
	seed.Start()

	for seed.Tail().Height() < 2 {
		time.Sleep(100 * time.Millisecond)
	}

	seed.Restart()

	for seed.Tail().Height() < 3 {
		time.Sleep(100 * time.Millisecond)
	}


}