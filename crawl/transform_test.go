package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestTransformJSON(t *testing.T) {
	c := check.New(t)

	b, err := transformJSON(nil, []byte(`{"A":     1234    }`))
	c.Nil(err)
	c.Equal(string(b), `{"A":1234}`)
}

func TestTransformSVG(t *testing.T) {
	c := check.New(t)

	b, err := transformSVG(nil, []byte(`<path fill="#ffffff"/>`))
	c.Nil(err)
	c.Equal(string(b), `<path fill="#fff"/>`)
}
