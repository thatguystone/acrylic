package crawl

import (
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
)

// Transform applies a single transform to the given content
type Transform func(cr *Crawler, c *Content, b []byte) ([]byte, error)

const (
	jsType   = "application/javascript"
	jsonType = "application/json"
	svgType  = "image/svg+xml"
)

var mini = minify.New()

func init() {
	mini.AddFunc(htmlType, html.Minify)
	mini.AddFunc(cssType, css.Minify)
	mini.AddFunc(jsType, js.Minify)
	mini.AddFunc(jsonType, json.Minify)
	mini.AddFunc(svgType, svg.Minify)
}

func transformJSON(cr *Crawler, c *Content, b []byte) ([]byte, error) {
	return mini.Bytes(jsonType, b)
}

func transformSVG(cr *Crawler, c *Content, b []byte) ([]byte, error) {
	return mini.Bytes(svgType, b)
}
