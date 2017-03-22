package acrylic

import (
	"errors"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thatguystone/cog/check"
)

func TestProxyBasic(t *testing.T) {
	c := check.New(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	c.Must.Nil(err)
	defer l.Close()

	p := Proxy{
		To: "http://" + l.Addr().String(),
	}

	err = <-p.pollReady(10 * time.Millisecond)
	c.Nil(err)
}

func TestProxyNotReady(t *testing.T) {
	c := check.New(t)

	p := Proxy{
		To: "http://127.0.0.1:999999",
	}

	err := <-p.pollReady(10 * time.Millisecond)
	c.NotNil(err)

	w := httptest.NewRecorder()
	p.Serve(nil, w, httptest.NewRequest("GET", "/", nil))
	c.Equal(w.Code, 502)

	w = httptest.NewRecorder()
	p.Serve(errors.New("error"), w, httptest.NewRequest("GET", "/", nil))
	c.Equal(w.Code, 500)
	c.Contains(w.Body.String(), "Error")
}
