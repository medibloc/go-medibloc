package logging

import "github.com/sirupsen/logrus"

// WithField creates an entry and adds a field to it. If you want multiple
// fields, use `WithFields`.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithField(key string, value interface{}) *logrus.Entry {
	return vLog().WithField(key, value)
}

// WithFields creates an entry and adds multiple fields to it. This is simply
// a helper for `WithField`, invoking it once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithFields(fields logrus.Fields) *logrus.Entry {
	return vLog().WithFields(fields)
}

// WithError creates an entry and adds an error to it, using the value defined
// in ErrorKey as key.
func WithError(err error) *logrus.Entry {
	return vLog().WithError(err)
}

// Debugf logs a message at level Debug on the verbose logger.
func Debugf(format string, args ...interface{}) {
	vLog().Debugf(format, args)
}

// Infof logs a message at level Info on the verbose logger.
func Infof(format string, args ...interface{}) {
	vLog().Infof(format, args)
}

// Printf logs a message at level Info on the verbose logger.
func Printf(format string, args ...interface{}) {
	vLog().Printf(format, args)
}

// Warnf logs a message at level Warn on the verbose logger.
func Warnf(format string, args ...interface{}) {
	vLog().Warnf(format, args)
}

// Warningf logs a message at level Warn on the verbose logger.
func Warningf(format string, args ...interface{}) {
	vLog().Warningf(format, args)
}

// Errorf logs a message at level Error on the verbose logger.
func Errorf(format string, args ...interface{}) {
	vLog().Errorf(format, args)
}

// Fatalf logs a message at level Fatal on the verbose logger.
func Fatalf(format string, args ...interface{}) {
	vLog().Fatalf(format, args)
}

// Panicf logs a message at level Panic on the verbose logger.
func Panicf(format string, args ...interface{}) {
	vLog().Panicf(format, args)
}

// Debug logs a message at level Debug on the verbose logger.
func Debug(args ...interface{}) {
	vLog().Debug(args)
}

// Info logs a message at level Info on the verbose logger.
func Info(args ...interface{}) {
	vLog().Info(args)
}

// Print logs a message at level Info on the verbose logger.
func Print(args ...interface{}) {
	vLog().Print(args)
}

// Warn logs a message at level Warn on the verbose logger.
func Warn(args ...interface{}) {
	vLog().Warn(args)
}

// Warning logs a message at level Warn on the verbose logger.
func Warning(args ...interface{}) {
	vLog().Warning(args)
}

// Error logs a message at level Error on the verbose logger.
func Error(args ...interface{}) {
	vLog().Error(args)
}

// Fatal logs a message at level Fatal on the verbose logger.
func Fatal(args ...interface{}) {
	vLog().Fatal(args)
}

// Panic logs a message at level Panic on the verbose logger.
func Panic(args ...interface{}) {
	vLog().Panic(args)
}

// Debugln logs a message at level Debug on the verbose logger.
func Debugln(args ...interface{}) {
	vLog().Debugln(args)
}

// Infoln logs a message at level Info on the verbose logger.
func Infoln(args ...interface{}) {
	vLog().Infoln(args)
}

// Println logs a message at level Info on the verbose logger.
func Println(args ...interface{}) {
	vLog().Println(args)
}

// Warnln logs a message at level Warn on the verbose logger.
func Warnln(args ...interface{}) {
	vLog().Warnln(args)
}

// Warningln logs a message at level Warn on the verbose logger.
func Warningln(args ...interface{}) {
	vLog().Warningln(args)
}

// Errorln logs a message at level Error on the verbose logger.
func Errorln(args ...interface{}) {
	vLog().Errorln(args)
}

// Fatalln logs a message at level Fatal on the verbose logger.
func Fatalln(args ...interface{}) {
	vLog().Fatalln(args)
}

// Panicln logs a message at level Panic on the verbose logger.
func Panicln(args ...interface{}) {
	vLog().Panicln(args)
}
