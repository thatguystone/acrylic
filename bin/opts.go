package bin

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

// Exts set the file extensions to watch for changes
func Exts(exts ...string) Option {
	return option(func(b *bin) {
		b.exts = exts
	})
}

// LogTo sets the log function
func LogTo(cb func(string, ...interface{})) Option {
	return option(func(b *bin) {
		b.logf = cb
	})
}
