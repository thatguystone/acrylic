package imgscale

import "fmt"

// An Option is passed to New() to change default options
type Option interface {
	applyTo(s *scaler)
}

type option func(s *scaler)

func (o option) applyTo(s *scaler) { o(s) }

// Root sets the root path for finding images from req.URL.Path
func Root(path string) Option {
	return option(func(s *scaler) {
		s.root = path
	})
}

// Cache sets the path to the image cache
func Cache(path string) Option {
	return option(func(s *scaler) {
		s.cache = path
	})
}

// MaxSubprocs sets the maximum number of scaler processses to run at the same
// time
func MaxSubprocs(n int) Option {
	if n <= 0 {
		panic(fmt.Errorf("invalid max subprocs: %q <= 0", n))
	}

	return option(func(s *scaler) {
		s.sema = make(chan struct{}, n)
	})
}
