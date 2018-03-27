package logging

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

// NewFileRotateHooker returns file rotate hooker.
func NewFileRotateHooker(path string, age uint32) logrus.Hook {
	if path == "" {
		panic("Failed to parse logger folder:" + path + ".")
	}
	if !filepath.IsAbs(path) {
		path, _ = filepath.Abs(path)
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		panic("Failed to create logger folder:" + path + ". err:" + err.Error())
	}

	filePath := path + "/medibloc-%Y%m%d%H.log"
	linkPath := path + "/medibloc.log"

	options := []rotatelogs.Option{
		rotatelogs.WithLinkName(linkPath),
		rotatelogs.WithRotationTime(time.Hour),
	}
	if age > 0 {
		options = append(options, rotatelogs.WithMaxAge(time.Duration(age)*time.Second))
	}
	writer, err := rotatelogs.New(
		filePath,
		options...,
	)
	if err != nil {
		panic("Failed to create rotate logs. err:" + err.Error())
	}

	writerMap := make(lfshook.WriterMap)
	for _, level := range logrus.AllLevels {
		writerMap[level] = writer
	}
	return lfshook.NewHook(writerMap, nil)
}
