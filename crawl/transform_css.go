package crawl

import (
	"regexp"
	"strings"
)

type cssTransform struct {
	css     string
	matches []cssMatch
}

type cssMatch struct {
	orig string
	url  string
	link ResolvedLinker
}

var (
	reCSSURL    = regexp.MustCompile(`url\(["']?(.*?)["']?\)`)
	reCSSImport = regexp.MustCompile(`@import ["'](.*?)["']`)
)

func transformCSS(lr LinkResolver, b []byte) ([]byte, error) {
	b = newCSSTransform(lr, string(b)).get()

	// Minify last so that any quotes added from url() replacements will be
	// removed if possible
	return Minify.Bytes(cssType, b)
}

func newCSSTransform(lr LinkResolver, css string) (tf cssTransform) {
	tf.css = css
	tf.extract(lr, reCSSURL)
	tf.extract(lr, reCSSImport)
	return
}

func (tf *cssTransform) extract(lr LinkResolver, re *regexp.Regexp) {
	for _, m := range re.FindAllStringSubmatch(tf.css, -1) {
		tf.matches = append(tf.matches, cssMatch{
			orig: m[0],
			url:  m[1],
			link: lr.ResolveLink(m[1]),
		})
	}
}

func (tf cssTransform) get() []byte {
	replaces := make([]string, 0, len(tf.matches)*2)

	for _, match := range tf.matches {
		rel := match.link.Get()

		if match.url != rel {
			replaces = append(replaces,
				match.orig, strings.Replace(match.orig, match.url, rel, 1))
		}
	}

	out := strings.NewReplacer(replaces...).Replace(tf.css)
	return []byte(out)
}
