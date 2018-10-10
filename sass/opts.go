package sass

import (
	"github.com/thatguystone/acrylic"
	"github.com/thatguystone/acrylic/watch"
)

// An Option is passed to New() to change default options
type Option interface {
	applyTo(s *sass)
}

type option func(s *sass)

func (o option) applyTo(s *sass) { o(s) }

// Entry adds another entry point
func Entry(path string) Option {
	return option(func(s *sass) {
		s.entries = append(s.entries, path)
	})
}

// IncludePaths adds paths to sass's include paths
func IncludePaths(paths ...string) Option {
	return option(func(s *sass) {
		s.includePaths = append(s.includePaths, paths...)
	})
}

// LogTo sets the log function
func LogTo(log acrylic.Logger) Option {
	return option(func(s *sass) {
		s.log = log
	})
}

// Watch attaches the sass instance to the given watch
func Watch(w *watch.Watch) Option {
	return option(func(s *sass) {
		w.Notify(s)
	})
}
