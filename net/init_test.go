package net

import (
	"testing"

	"os"

	"github.com/medibloc/go-medibloc/util/logging"
)

func TestMain(m *testing.M) {
	logging.TestHook()
	os.Exit(m.Run())
}
