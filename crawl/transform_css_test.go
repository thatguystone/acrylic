package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestTransformCSSRewrites(t *testing.T) {
	c := check.New(t)

	const css = `` +
		`@import "/img.gif";` +
		`.body1 { background: url(/img.gif); }` +
		`.body2 { background: url("/img.gif"); }` +
		`.body3 { background: url('/img.gif'); }`

	lr := linkRewrite{
		"/img.gif": "img.hash.gif",
	}

	out, err := transformCSS(lr, []byte(css))
	c.Nil(err)
	c.Contains(string(out), "img.hash.gif")
	c.NotContains(string(out), "img.gif")
}

func TestTransformCSSError(t *testing.T) {
	c := check.New(t)

	const css = `}`
	lr := linkRewrite{}

	_, err := transformCSS(lr, []byte(css))
	c.NotNil(err)
}
