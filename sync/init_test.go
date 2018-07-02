package sync

import (
	"testing"

	"os"

	"github.com/medibloc/go-medibloc/util/logging"
)

func TestMain(m *testing.M) {
	logging.SetNullLogger()
	os.Exit(m.Run())
}
