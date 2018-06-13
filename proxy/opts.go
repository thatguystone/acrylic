package proxy

import "log"

// An Option is passed to New() to change default options
type Option interface {
	applyTo(p *Proxy)
}

type option func(p *Proxy)

func (o option) applyTo(p *Proxy) { o(p) }

// ErrorLog is a helper that sets the reverse proxy's ErrorLog
func ErrorLog(cb func(...interface{})) Option {
	return option(func(p *Proxy) {
		p.ErrorLog = log.New(&logRedirector{cb: cb}, "", 0)
	})
}

type logRedirector struct {
	cb func(...interface{})
}

func (w *logRedirector) Write(b []byte) (int, error) {
	w.cb(string(b))
	return len(b), nil
}
