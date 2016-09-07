package acrylic

import (
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
)

var Minify = minify.New()

func init() {
	Minify.AddFunc("text/css", css.Minify)
	Minify.AddFunc("text/html", html.Minify)
	Minify.AddFunc("text/javascript", js.Minify)
}
