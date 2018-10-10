package webpack

// An Option is passed to New() to change default options
type Option interface {
	applyTo(wp *webpack)
}

type option func(wp *webpack)

func (o option) applyTo(wp *webpack) { o(wp) }

// Bin changes the default bin path from "./node_modules/.bin/webpack-dev-server"
func Bin(path string) Option {
	return option(func(wp *webpack) {
		wp.bin = path
	})
}

// Port changes the default port from 9779
func Port(port uint16) Option {
	return option(func(wp *webpack) {
		wp.port = port
	})
}

// Args sets extra args to pass to webpack-dev-server
func Args(args ...string) Option {
	return option(func(wp *webpack) {
		wp.args = args
	})
}
