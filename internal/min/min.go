package min

import (
	"io"

	"github.com/tdewolff/minify"
	min_css "github.com/tdewolff/minify/css"
	min_html "github.com/tdewolff/minify/html"
	min_js "github.com/tdewolff/minify/js"
)

var min = minify.New()

func init() {
	min.AddFunc("text/css", min_css.Minify)
	min.AddFunc("text/html", min_html.Minify)
	min.AddFunc("text/javascript", min_js.Minify)
}

func Minify(mime string, w io.Writer, r io.Reader) error {
	return min.Minify(mime, w, r)
}
