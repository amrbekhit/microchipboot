package microchipboot

type logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
}

type nullLogger struct{}

func (l *nullLogger) Debugf(format string, args ...interface{}) {}
func (l *nullLogger) Infof(format string, args ...interface{})  {}

// The package logger
var pkgLog logger = &nullLogger{}

// SetLogger sets the logger used internally by the package.
func SetLogger(l logger) {
	pkgLog = l
}
