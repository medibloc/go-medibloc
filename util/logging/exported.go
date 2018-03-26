package logging

import "github.com/sirupsen/logrus"

func WithField(key string, value interface{}) *logrus.Entry {
	return vLog().WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return vLog().WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	return vLog().WithError(err)
}

func Debugf(format string, args ...interface{}) {
	vLog().Debugf(format, args)
}

func Infof(format string, args ...interface{}) {
	vLog().Infof(format, args)
}

func Printf(format string, args ...interface{}) {
	vLog().Printf(format, args)
}

func Warnf(format string, args ...interface{}) {
	vLog().Warnf(format, args)
}

func Warningf(format string, args ...interface{}) {
	vLog().Warningf(format, args)
}

func Errorf(format string, args ...interface{}) {
	vLog().Errorf(format, args)
}

func Fatalf(format string, args ...interface{}) {
	vLog().Fatalf(format, args)
}

func Panicf(format string, args ...interface{}) {
	vLog().Panicf(format, args)
}

func Debug(args ...interface{}) {
	vLog().Debug(args)
}

func Info(args ...interface{}) {
	vLog().Info(args)
}

func Print(args ...interface{}) {
	vLog().Print(args)
}

func Warn(args ...interface{}) {
	vLog().Warn(args)
}

func Warning(args ...interface{}) {
	vLog().Warning(args)
}

func Error(args ...interface{}) {
	vLog().Error(args)
}

func Fatal(args ...interface{}) {
	vLog().Fatal(args)
}

func Panic(args ...interface{}) {
	vLog().Panic(args)
}

func Debugln(args ...interface{}) {
	vLog().Debugln(args)
}

func Infoln(args ...interface{}) {
	vLog().Infoln(args)
}

func Println(args ...interface{}) {
	vLog().Println(args)
}

func Warnln(args ...interface{}) {
	vLog().Warnln(args)
}

func Warningln(args ...interface{}) {
	vLog().Warningln(args)
}

func Errorln(args ...interface{}) {
	vLog().Errorln(args)
}

func Fatalln(args ...interface{}) {
	vLog().Fatalln(args)
}

func Panicln(args ...interface{}) {
	vLog().Panicln(args)
}
