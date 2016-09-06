package errs

import (
	"testing"

	"github.com/thatguystone/cog/check/chlog"
)

func TestBasic(t *testing.T) {
	c, log := chlog.New(t)
	errs := New(log.Get(""))

	c.True(errs.Ok())

	errs.Errorf("test file", "this failed: %d", 123)
	c.False(errs.Ok())
}
