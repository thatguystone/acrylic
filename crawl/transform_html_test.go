package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestTransformHTMLRewrites(t *testing.T) {
	c := check.New(t)

	const html = `` +
		`<style>body { background: url(/img.gif); }</style>` +
		`<a href="/img.gif"></a>` +
		`<img src="/img.gif" srcset="/img.gif, /img.gif 2x">` +
		`<div style="background: url(/img.gif)"></div>`

	lr := linkRewrite{
		"/img.gif": "img.hash.gif",
	}

	out, err := transformHTML(lr, []byte(html))
	c.Nil(err)
	c.Contains(string(out), "img.hash.gif")
	c.NotContains(string(out), "img.gif")
}
