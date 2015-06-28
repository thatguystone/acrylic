package acrylib

import (
	"bytes"

	"github.com/russross/blackfriday"
)

type renderPage struct{}
type renderHTML struct{ renderPage }
type renderMarkdown struct{ renderPage }

type mdHtmlRenderer struct {
	blackfriday.Renderer
	err error
}

// Pages are always rendered anyway...
func (renderPage) alwaysRender() bool { return true }

func (renderHTML) renders(ext string) bool         { return ext == ".html" }
func (renderHTML) render(b []byte) ([]byte, error) { return b, nil }

func (renderMarkdown) renders(ext string) bool {
	switch ext {
	case ".markdown", ".md", ".mdown":
		return true
	}

	return false
}

func (renderMarkdown) render(b []byte) ([]byte, error) {
	renderer := &mdHtmlRenderer{
		Renderer: blackfriday.HtmlRenderer(blackfriday.HTML_USE_XHTML|
			blackfriday.HTML_USE_SMARTYPANTS|
			blackfriday.HTML_SMARTYPANTS_FRACTIONS|
			blackfriday.HTML_SMARTYPANTS_LATEX_DASHES,
			"",
			""),
	}

	out := blackfriday.MarkdownOptions(b, renderer, blackfriday.Options{
		Extensions: blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
			blackfriday.EXTENSION_TABLES |
			blackfriday.EXTENSION_FENCED_CODE |
			blackfriday.EXTENSION_AUTOLINK |
			blackfriday.EXTENSION_STRIKETHROUGH |
			blackfriday.EXTENSION_SPACE_HEADERS |
			blackfriday.EXTENSION_HEADER_IDS |
			blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
			blackfriday.EXTENSION_DEFINITION_LISTS,
	})

	return bytes.TrimSpace(out), renderer.err
}

func (mdr *mdHtmlRenderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	mdr.err = highlight(lang, out, text)
}
