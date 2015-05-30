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

type renderBase struct{}
type renderHtmlOut struct{ renderBase }
type renderSameExtOut struct{ renderBase }

type renderMarkdown struct{ renderHtmlOut }
type renderPlainText struct{ renderSameExtOut }
type renderJsCss struct{ renderSameExtOut }
type renderPassthru struct{ renderSameExtOut }

var renderers = []renderer{
	renderMarkdown{},
	renderPlainText{},
	renderJsCss{},
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

func (renderBase) renderable() bool            { return true }
func (renderHtmlOut) ext(c *content) string    { return ".html" }
func (renderSameExtOut) ext(c *content) string { return filepath.Ext(c.f.srcPath) }
func (renderBase) render(c *content) ([]byte, error) {
	panic("render not implemented! that's programmer error.")
}

func (renderMarkdown) render(c *content) ([]byte, error) {
	return bytes.TrimSpace(blackfriday.MarkdownCommon(c.rawContent)), nil
}
func (renderMarkdown) handles(ext string) bool {
	switch ext {
	case ".md", ".markdown", ".mdown":
		return true
	}

	return false
}

func (renderPlainText) render(c *content) ([]byte, error) {
	return c.rawContent, nil
}
func (renderPlainText) handles(ext string) bool {
	switch ext {
	case ".html", ".txt":
		return true
	}

	return false
}

func (renderJsCss) renderable() bool { return false }
func (renderJsCss) handles(ext string) bool {
	switch ext {
	case ".js", ".css":
		return true
	}

	return false
}

func (renderPassthru) renderable() bool        { return false }
func (renderPassthru) handles(ext string) bool { return true }
