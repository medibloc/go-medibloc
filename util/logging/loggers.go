package logging

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type emptyWriter struct{}

func (ew emptyWriter) Write(p []byte) (int, error) {
	return 0, nil
}

var (
	clog *logrus.Logger
	vlog *logrus.Logger
)

// Console returns console logger.
func Console() *logrus.Logger {
	if clog == nil {
		Init("/tmp", "info", 0)
	}
	return clog
}

func vLog() *logrus.Logger {
	if vlog == nil {
		Init("/tmp", "info", 0)
	}
	return vlog
}

// Init loggers.
func Init(path string, level string, age uint32) {
	levelNo, err := logrus.ParseLevel(level)
	if err != nil {
		panic("Invald log level:" + level)
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

	vLog().WithFields(logrus.Fields{
		"path":  path,
		"level": level,
	}).Info("Logger Configuration.")
}

// TestHook returns hook for testing log entry.
func TestHook() *test.Hook {
	logger, hook := test.NewNullLogger()
	clog = logger
	vlog = logger
	return hook
}
