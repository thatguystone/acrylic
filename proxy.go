package acrylic

import (
	"fmt"
	"html"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/thatguystone/cog"
)

// Proxy implements a reverse proxy
type Proxy struct {
	To    string // Where to proxy requests to
	once  sync.Once
	url   *url.URL
	proxy *httputil.ReverseProxy
}

func (p *Proxy) init() {
	p.once.Do(func() {
		var err error

		p.url, err = url.Parse(p.To)
		cog.Must(err, "[proxy] failed to parse to=`%s`: %v", p.To, err)

		p.proxy = httputil.NewSingleHostReverseProxy(p.url)
	})
}

// pollReady polls the backend server until it connects or gives up. If it
// connects, the returned channel is closed, otherwise, a timeout error is sent.
func (p *Proxy) pollReady(wait time.Duration) <-chan error {
	ch := make(chan error)

	go func() {
		giveUp := time.After(wait)

		for !p.isReady() {
			select {
			case <-giveUp:
				ch <- fmt.Errorf(
					"[proxy] could not reach `%s` in a reasonable amount of time",
					p.url)
				return

			case <-time.After(time.Millisecond):
			}
		}

		close(ch)
	}()

	return ch
}

func (p *Proxy) isReady() (ready bool) {
	p.init()

	conn, err := net.DialTimeout("tcp", p.url.Host, 100*time.Millisecond)
	if err == nil {
		ready = true
		conn.Close()
	}

	return
}

// ServeHTTP implements http.Handler
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.init()
	p.proxy.ServeHTTP(w, r)
}

func (p *Proxy) Serve(err error, w http.ResponseWriter, r *http.Request) {
	if err != nil {
		sendError(err, w)
	} else {
		p.ServeHTTP(w, r)
	}
}

func sendError(err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusInternalServerError)

	fmt.Fprintf(w,
		`
			<style>
				body {
					background: #252830;
					color: #fff;
				}
			</style>
			<h1>Error</h1><pre>%s</pre>
		`,
		html.EscapeString(err.Error()))
}
