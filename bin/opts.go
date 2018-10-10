package bin

import (
	"github.com/thatguystone/acrylic"
	"github.com/thatguystone/acrylic/watch"
)

// An Option is passed to New() to change default options
type Option interface {
	applyTo(b *bin)
}

type option func(b *bin)

func (o option) applyTo(b *bin) { o(b) }

// BuildCmd sets the command to execute after a change is detected
func BuildCmd(cmd ...string) Option {
	return option(func(b *bin) {
		b.buildCmd = cmd
	})
}

// LogTo sets the log function
func LogTo(log acrylic.Logger) Option {
	return option(func(b *bin) {
		b.log = log
	})
}

// Watch attaches the bin instance to the given watch
func Watch(w *watch.Watch) Option {
	return option(func(b *bin) {
		w.Notify(b)
	})
}

// Exts sets the file extensions to watch for changes
func Exts(exts ...string) Option {
	return option(func(b *bin) {
		b.exts = exts
	})
}
