package medlet

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

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
	assert.True(t, proto.Equal(pb, defaultConfig()))
}

func TestDefaultConfig(t *testing.T) {
	path := filepath.Join("testdata", t.Name()+".golden")
	if *update {
		ioutil.WriteFile(path, []byte(defaultConfigString()), 0644)
	}

	golden, _ := ioutil.ReadFile(path)
	assert.Equal(t, string(golden), defaultConfigString())
}
