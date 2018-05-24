package acrylic

import (
	"fmt"
	"net"
	"net/http/httputil"
	"net/url"
	"time"
)

// A Proxy wraps a ReverseProxy with some utility functions
type Proxy struct {
	*httputil.ReverseProxy
	url *url.URL
}

// NewProxy creates a new Proxy wrapper
func NewProxy(target string) (*Proxy, error) {
	url, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to parse %q: %v", url, err)
	}

	p := &Proxy{
		url:          url,
		ReverseProxy: httputil.NewSingleHostReverseProxy(url),
	}

	return p, nil
}

// PollReady polls the backend server until it connects or gives up. If it
// connects, the returned channel is closed, otherwise, a timeout error is
// sent.
func (p *Proxy) PollReady(wait time.Duration) <-chan error {
	ch := make(chan error)

	go func() {
		defer close(ch)

		timeout := time.After(wait)

		for !p.isReady() {
			select {
			case <-timeout:
				ch <- fmt.Errorf("proxy: could not reach %q after %s", p.url, wait)
				return

			case <-time.After(time.Millisecond):
			}
		}
	}()

	return ch
}

func (p *Proxy) isReady() (ready bool) {
	conn, err := net.DialTimeout("tcp", p.url.Host, 100*time.Millisecond)
	if err == nil {
		ready = true
		conn.Close()
	}

	return
}
