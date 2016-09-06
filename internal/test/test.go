package test

import (
	"testing"

	"github.com/thatguystone/acrylic/internal/config"
	"github.com/thatguystone/acrylic/internal/errs"
	"github.com/thatguystone/acrylic/internal/state"
	"github.com/thatguystone/acrylic/internal/unused"
	"github.com/thatguystone/cog/check"
	"github.com/thatguystone/cog/check/chlog"
	"github.com/thatguystone/cog/clog"
)

type C struct {
	*check.C
	Log *clog.Log
	St  *state.S
}

var GifBin = []byte{
	0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
	0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
	0x00, 0x3b,
}

func New(t *testing.T) *C {
	c, log := chlog.New(t)

	return &C{
		C:   c,
		Log: log,
		St: &state.S{
			Cfg:    config.New().InDir(c.FS.Path("")),
			Errs:   errs.New(log.Get("errs")),
			Unused: unused.New(),
		},
	}
}
