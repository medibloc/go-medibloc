package logging

import (
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type functionHooker struct{}

func NewFunctionHooker() logrus.Hook {
	return &functionHooker{}
}

func (h *functionHooker) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *functionHooker) Fire(entry *logrus.Entry) error {
	pc := make([]uintptr, 10)
	runtime.Callers(6, pc)
	for i := 0; i < 10; i++ {
		if pc[i] == 0 {
			break
		}
		f := runtime.FuncForPC(pc[i])
		file, line := f.FileLine(pc[i])
		if isLogrusFunc(file) {
			continue
		}
		entry.Data["line"] = line
		entry.Data["func"] = path.Base(f.Name())
		entry.Data["file"] = filepath.Base(file)
		break
	}
	return nil
}

func isLogrusFunc(file string) bool {
	return strings.Contains(file, "sirupsen")
}
