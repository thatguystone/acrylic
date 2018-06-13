package proxy

import (
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

	p, err := New(
		"http://"+l.Addr().String(),
		ErrorLog(c.Log))
	c.Must.Nil(err)

	err = <-p.PollReady(1 * time.Millisecond)
	c.Nil(err)
}

func TestProxyNotReady(t *testing.T) {
	c := check.New(t)

	p, err := New(
		"http://127.0.0.1:999999",
		ErrorLog(c.Log))
	c.Must.Nil(err)

	err = <-p.PollReady(1 * time.Millisecond)
	c.NotNil(err)

	w := httptest.NewRecorder()
	p.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	c.Equal(w.Code, 502)
}

func TestProxyURLParseError(t *testing.T) {
	c := check.New(t)

	_, err := New(`%20://`)
	c.Must.NotNil(err)
	c.Contains(err.Error(), "failed to parse")
}
