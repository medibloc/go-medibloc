package sync

import (
	"testing"

	"github.com/medibloc/go-medibloc/util/logging"
)

func TestMain(m *testing.M) {
	logging.TestHook()
	m.Run()
}
