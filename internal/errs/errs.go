package errs

import (
	"fmt"

	"github.com/thatguystone/cog/clog"
)

type E struct {
	failed bool
	log    *clog.Logger
}

func New(log *clog.Logger) *E {
	return &E{
		log: log,
	}
}

func (e *E) Errorf(file, format string, args ...interface{}) {
	e.failed = true
	e.log.Errorf("with file %s: %s", fmt.Sprintf(format, args...))
}

func (e *E) Ok() bool {
	return !e.failed
}
