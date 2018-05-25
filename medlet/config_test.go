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

package medlet

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden files")

func init() {
	flag.Parse()
}

func TestConfigNotExist(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "config")
	require.Nil(t, err)
	os.Remove(f.Name())

	pb := LoadConfig(f.Name())
	defer os.Remove(f.Name())
	assert.True(t, proto.Equal(pb, DefaultConfig()))
}

func TestDefaultConfig(t *testing.T) {
	path := filepath.Join("testdata", strings.ToLower(t.Name())+".golden")
	if *update {
		ioutil.WriteFile(path, []byte(defaultConfigString()), 0644)
	}

	golden, _ := ioutil.ReadFile(path)
	assert.Equal(t, string(golden), defaultConfigString())
}
