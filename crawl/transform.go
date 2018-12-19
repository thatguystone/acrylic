package crawl

import (
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
)

// Transform applies a single transform to the given content
type Transform func(lr LinkResolver, b []byte) ([]byte, error)

var (
	// Minify is the minifier the crawler uses
	Minify = minify.New()

	defaultTransforms = map[string][]Transform{
		htmlType: {transformHTML},
		cssType:  {transformCSS},
		jsonType: {transformJSON},
		svgType:  {transformSVG},
	}
)

func init() {
	Minify.AddFunc(htmlType, html.Minify)
	Minify.AddFunc(cssType, css.Minify)
	Minify.AddFunc(jsType, js.Minify)
	Minify.AddFunc(jsonType, json.Minify)
	Minify.AddFunc(svgType, svg.Minify)
}

func transformJSON(lr LinkResolver, b []byte) ([]byte, error) {
	return Minify.Bytes(jsonType, b)
}

func transformSVG(lr LinkResolver, b []byte) ([]byte, error) {
	return Minify.Bytes(svgType, b)
}
