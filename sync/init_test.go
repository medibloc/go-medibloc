package sync

import (
	"testing"

	"github.com/medibloc/go-medibloc/util/logging"
	"os"
)

func TestMain(m *testing.M) {
	logging.TestHook()
	os.Exit(m.Run())
}
