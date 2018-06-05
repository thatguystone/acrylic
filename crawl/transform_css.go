package crawl

import (
	"regexp"
	"strings"
)

type cssTransform struct {
	c       *Content
	css     string
	matches []cssMatch
}

type cssMatch struct {
	orig    string
	url     string
	linkRes LinkResolver
}

const cssType = "text/css"

var (
	reCSSURL    = regexp.MustCompile(`url\(["']?(.*?)["']?\)`)
	reCSSImport = regexp.MustCompile(`@import ["'](.*?)["']`)
)

func transformCSS(cr *Crawler, c *Content, b []byte) ([]byte, error) {
	b = newCSSTransform(cr, c, string(b)).get()

	// Minify last so that any quotes added from url() replacements will be
	// removed if possible
	return mini.Bytes(cssType, b)
}

func newCSSTransform(cr *Crawler, c *Content, css string) (tf cssTransform) {
	tf.c = c
	tf.css = css
	tf.extract(cr, c, reCSSURL)
	tf.extract(cr, c, reCSSImport)
	return
}

func (tf *cssTransform) extract(cr *Crawler, c *Content, re *regexp.Regexp) {
	for _, m := range re.FindAllStringSubmatch(tf.css, -1) {
		tf.matches = append(tf.matches, cssMatch{
			orig:    m[0],
			url:     m[1],
			linkRes: cr.ResolveLink(c, m[1]),
		})
	}
}

func (tf cssTransform) get() []byte {
	replaces := make([]string, 0, len(tf.matches)*2)

	for _, match := range tf.matches {
		rel := match.linkRes.Get()

		if match.url != rel {
			replaces = append(replaces,
				match.orig, strings.Replace(match.orig, match.url, rel, 1))
		}
	}

	out := strings.NewReplacer(replaces...).Replace(tf.css)
	return []byte(out)
}
