package logging_test

import (
	"os"
	"testing"

	log "github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"path/filepath"
)

func TestLogger(t *testing.T) {
	tmp := filepath.Join(os.TempDir(), "medi_test", t.Name())
	defer os.RemoveAll(tmp)
	log.Init(tmp, "debug", 0)
	log.Info("Log example")
	log.WithFields(logrus.Fields{
		"type": "example",
		"err":  "Error",
	}).Error("Error example")
	log.Console().WithFields(logrus.Fields{
		"output": "console & file",
		"err":    "nil",
	}).Info("Console example")
}

func TestHook(t *testing.T) {
	hook := log.TestHook()
	log.Info("Test Hook")
	assert.Equal(t, 1, len(hook.Entries))
	assert.Regexp(t, ".*Test Hook.*", hook.LastEntry().Message)
}
