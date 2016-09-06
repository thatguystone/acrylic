package state

import (
	"github.com/thatguystone/acrylic/internal/config"
	"github.com/thatguystone/acrylic/internal/errs"
	"github.com/thatguystone/acrylic/internal/pool"
	"github.com/thatguystone/acrylic/internal/unused"
)

type S struct {
	Cfg    *config.C
	Errs   *errs.E
	Run    *pool.Runner
	Unused *unused.U
}
