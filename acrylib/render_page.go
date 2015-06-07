package acrylib

import (
	"bytes"

	"github.com/russross/blackfriday"
)

type renderPage struct{}
type renderHTML struct{ renderPage }
type renderMarkdown struct{ renderPage }

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
	return bytes.TrimSpace(blackfriday.MarkdownCommon(b)), nil
}
