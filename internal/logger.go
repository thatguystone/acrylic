package internal

import (
	"github.com/thatguystone/acrylic"
)

type logger struct {
	prefix string
	logf   LogFunc
}

// LogFunc is the function called for everything
type LogFunc func(format string, a ...interface{})

// NewLogger creates a new acrylic.Logger that pushes everything to the given
// LogFun with the given prefix.
func NewLogger(prefix string, logf LogFunc) acrylic.Logger {
	return &logger{
		prefix: prefix,
		logf:   logf,
	}
}

func (l *logger) Log(msg string) {
	l.logf("I: %s: %s", l.prefix, msg)
}

func (l *logger) Error(err error, msg string) {
	l.logf("E: %s: %s: %v", l.prefix, msg, err)
}
