package imgscale

import (
	"fmt"

	"github.com/thatguystone/acrylic/internal/cache"
)

// An Option is passed to New() to change default options
type Option interface {
	applyTo(isc *imgscale)
}

type option func(isc *imgscale)

func (o option) applyTo(isc *imgscale) { o(isc) }

// Root sets the root path for finding images from req.URL.Path
func Root(path string) Option {
	return option(func(isc *imgscale) {
		isc.root = path
	})
}

// Cache sets the path to the image cache
func Cache(path string) Option {
	return option(func(isc *imgscale) {
		isc.cache = cache.NewReadThrough(path)
	})
}

// MaxSubprocs sets the maximum number of scaler processes to run at the same
// time
func MaxSubprocs(n int) Option {
	if n <= 0 {
		panic(fmt.Errorf("invalid max subprocs: %q <= 0", n))
	}

	return option(func(isc *imgscale) {
		isc.sema = make(chan struct{}, n)
	})
}
