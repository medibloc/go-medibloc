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
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type functionHooker struct{}

// NewFunctionHooker adds function info to log entry.
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
