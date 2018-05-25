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

package logging

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type emptyWriter struct{}

func (ew emptyWriter) Write(p []byte) (int, error) {
	return 0, nil
}

var (
	mu   sync.Mutex
	clog *logrus.Logger
	vlog *logrus.Logger
)

// Console returns console logger.
func Console() *logrus.Logger {
	mu.Lock()
	defer mu.Unlock()

	if clog == nil {
		Init("/tmp", "info", 0)
	}
	return clog
}

func vLog() *logrus.Logger {
	mu.Lock()
	defer mu.Unlock()

	if vlog == nil {
		Init("/tmp", "info", 0)
	}
	return vlog
}

// Init loggers.
func Init(path string, level string, age uint32) {
	levelNo, err := logrus.ParseLevel(level)
	if err != nil {
		panic("Invalid log level:" + level)
	}

	fileHooker := NewFileRotateHooker(path, age)
	funcHooker := NewFunctionHooker()

	clog = logrus.New()
	clog.Hooks.Add(funcHooker)
	clog.Hooks.Add(fileHooker)
	clog.Out = os.Stdout
	clog.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	clog.Level = logrus.DebugLevel

	vlog = logrus.New()
	vlog.Hooks.Add(funcHooker)
	vlog.Hooks.Add(fileHooker)
	vlog.Out = &emptyWriter{}
	vlog.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	vlog.Level = levelNo

	vlog.WithFields(logrus.Fields{
		"path":  path,
		"level": level,
	}).Info("Logger Configuration.")
}

// TestHook returns hook for testing log entry.
func TestHook() *test.Hook {
	mu.Lock()
	defer mu.Unlock()

	logger, hook := test.NewNullLogger()
	clog = logger
	vlog = logger
	return hook
}
