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

package logging_test

import (
	"os"
	"testing"

	"path/filepath"

	log "github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
