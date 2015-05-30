package toner

import (
	"bytes"
	"path/filepath"

	"github.com/russross/blackfriday"
)

type renderer interface {
	handles(ext string) bool

	// If the content can be rendered
	renderable() bool

	// Get the extension for this content
	ext(c *content) string

	// Render the content
	render(c *content) ([]byte, error)
}

type renderMarkdown struct{}
type renderPassthru struct{}

var renderers = []renderer{
	renderMarkdown{},
	renderPassthru{},
}

func getRenderer(c *content) renderer {
	ext := filepath.Ext(c.f.srcPath)
	for _, r := range renderers {
		if r.handles(ext) {
			return r
		}
	}

	panic("could not find a renderer!")
}

func (renderMarkdown) handles(ext string) bool {
	switch ext {
	case ".md", ".markdown", ".mdown":
		return true
	default:
		return false
	}
}

func (renderMarkdown) renderable() bool {
	return true
}

func (renderMarkdown) ext(c *content) string {
	return ".html"
}

func (renderMarkdown) render(c *content) ([]byte, error) {
	return bytes.TrimSpace(blackfriday.MarkdownCommon(c.rawContent)), nil
}

func (renderPassthru) handles(ext string) bool {
	return true
}

func (renderPassthru) renderable() bool {
	return false
}

func (renderPassthru) ext(c *content) string {
	return filepath.Ext(c.f.srcPath)
}

func (renderPassthru) render(c *content) ([]byte, error) {
	panic("renderPassthru can't render anything!")
}
