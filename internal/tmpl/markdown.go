package tmpl

import (
	"github.com/flosch/pongo2"
	"github.com/russross/blackfriday"
)

func init() {
	pongo2.RegisterFilter("markdown", filterMarkdown)
}

func filterMarkdown(in *pongo2.Value, param *pongo2.Value) (
	out *pongo2.Value,
	err *pongo2.Error) {

	out = pongo2.AsSafeValue(string(blackfriday.MarkdownCommon([]byte(in.String()))))
	return
}
